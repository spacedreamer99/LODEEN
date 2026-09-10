#version 330 core
in vec3 vWorldPos;
in vec3 vNormal;
in vec2 vUV;
out vec4 FragColor;

uniform vec3 uLightPos;
uniform vec3 uLightColor;
uniform vec3 uCamPos;
uniform vec4 uBaseColorFactor;
uniform float uMetallic;
uniform float uRoughness;
uniform int uAlphaMode;
uniform float uAlphaCutoff;
uniform int uHasTexture;
uniform sampler2D uBaseColorTex;
uniform int uHasMRTex;
uniform sampler2D uMRTex;
uniform int uCamInside;

const float PI = 3.14159265359;

float distributionGGX(vec3 N, vec3 H, float r) {
    float a = r * r, a2 = a * a;
    float NdotH = max(dot(N, H), 0.0);
    float denom = NdotH * NdotH * (a2 - 1.0) + 1.0;
    return a2 / (PI * denom * denom);
}
float geometrySchlick(float NdotV, float r) {
    float k = ((r + 1.0) * (r + 1.0)) / 8.0;
    return NdotV / (NdotV * (1.0 - k) + k);
}
float geometrySmith(vec3 N, vec3 V, vec3 L, float r) {
    return geometrySchlick(max(dot(N, V), 0.0), r) *
           geometrySchlick(max(dot(N, L), 0.0), r);
}
vec3 fresnelSchlick(float cosT, vec3 F0) {
    return F0 + (1.0 - F0) * pow(1.0 - abs(cosT), 5.0);
}

void main() {
    vec4 baseColor = uBaseColorFactor;
    if (uHasTexture == 1) baseColor *= texture(uBaseColorTex, vUV);

    float alpha = (uAlphaMode == 0) ? 1.0 : baseColor.a;
    if (uAlphaMode == 1 && alpha < uAlphaCutoff) discard;
    if (uAlphaMode != 0 && alpha < 0.01) discard;

    float rough = uRoughness;
    float metal = uMetallic;
    if (uHasMRTex == 1) {
        vec4 mr = texture(uMRTex, vUV);
        rough *= mr.g;
        metal *= mr.b;
    }
    rough = clamp(rough, 0.04, 1.0);
    metal = clamp(metal, 0.0, 1.0);

    vec3 albedo = baseColor.rgb;
    vec3 N_orig = normalize(vNormal);
    vec3 V = normalize(uCamPos - vWorldPos);
    vec3 L = normalize(uLightPos - vWorldPos);
    vec3 H = normalize(V + L);

    // ============ ОБЛАКА / АТМОСФЕРА ============
    if (uAlphaMode == 2) {
        // Одна стенка: снаружи — front, изнутри — back.
        // Убирает наложение двух стенок и артефакты UV.
        bool wantFront = (uCamInside == 0);
        if (gl_FrontFacing != wantFront) discard;

        // Освещённость: до терминатора — 1, за ним — плавно в 0.
        float dayFactor = smoothstep(-0.10, 0.20, dot(N_orig, L));

        // Плотность облаков из alpha-канала текстуры.
        // Это ДИКТУЕТ прозрачность вне зависимости от дня/ночи —
        // поэтому ночью облака не пропускают звёзды.
        float coverage = alpha;

        // Но днём облака выглядят тоньше, потому что освещены с боков
        // и рассеивают свет — этот эффект даём через минимум прозрачности
        // только на дневной стороне.
        float minTransparency = mix(0.0, 0.35, dayFactor);
        float finalAlpha = max(coverage, minTransparency);
        // Гарантируем непрозрачность при плотной текстуре ночью
        if (dayFactor < 0.05) finalAlpha = coverage;

        if (finalAlpha < 0.01) discard;

        // Цвет: яркость зависит от дня/ночи
        vec3 N = N_orig;
        float NdotV = max(dot(N, V), 0.0);
        float NdotL = max(dot(N, L), 0.0);

        vec3 F0 = mix(vec3(0.04), albedo, metal);
        float NDF = distributionGGX(N, H, rough);
        float G = geometrySmith(N, V, L, rough);
        vec3 F = fresnelSchlick(dot(H, V), F0);
        vec3 specular = (NDF * G * F) / (4.0 * NdotV * NdotL + 0.0001);
        vec3 kD = (vec3(1.0) - F) * (1.0 - metal);
        vec3 color = (kD * albedo / PI + specular) * uLightColor * NdotL;

        // Ночь — почти чёрный. День — полный цвет.
        color *= dayFactor;

        FragColor = vec4(pow(color, vec3(1.0/2.2)), finalAlpha);
        return;
    }

    // ============ ОБЫЧНЫЕ МАТЕРИАЛЫ ============
    vec3 N = N_orig;
    if (!gl_FrontFacing) N = -N;
    float NdotV = max(dot(N, V), 0.0);
    float NdotL = max(dot(N, L), 0.0);

    vec3 F0 = mix(vec3(0.04), albedo, metal);
    float NDF = distributionGGX(N, H, rough);
    float G = geometrySmith(N, V, L, rough);
    vec3 F = fresnelSchlick(dot(H, V), F0);
    vec3 specular = (NDF * G * F) / (4.0 * NdotV * NdotL + 0.0001);
    vec3 kD = (vec3(1.0) - F) * (1.0 - metal);
    vec3 color = (kD * albedo / PI + specular) * uLightColor * NdotL;

    FragColor = vec4(pow(color, vec3(1.0/2.2)), alpha);
}
