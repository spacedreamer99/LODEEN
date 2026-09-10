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
    return F0 + (1.0 - F0) * pow(1.0 - cosT, 5.0);
}

void main() {
    vec4 baseColor = uBaseColorFactor;
    if (uHasTexture == 1) baseColor *= texture(uBaseColorTex, vUV);

    float alpha = (uAlphaMode == 0) ? 1.0 : baseColor.a;
    if (uAlphaMode == 1 && alpha < uAlphaCutoff) discard;
    if (alpha < 0.01) discard;

    vec3 albedo = baseColor.rgb;
    vec3 N = normalize(vNormal);
    vec3 V = normalize(uCamPos - vWorldPos);
    vec3 L = normalize(uLightPos - vWorldPos);
    vec3 H = normalize(V + L);

    vec3 F0 = mix(vec3(0.04), albedo, uMetallic);
    float NDF = distributionGGX(N, H, uRoughness);
    float G = geometrySmith(N, V, L, uRoughness);
    vec3 F = fresnelSchlick(max(dot(H, V), 0.0), F0);

    vec3 specular = (NDF * G * F) /
        (4.0 * max(dot(N, V), 0.0) * max(dot(N, L), 0.0) + 0.0001);
    vec3 kD = (vec3(1.0) - F) * (1.0 - uMetallic);
    float NdotL = max(dot(N, L), 0.0);
    vec3 Lo = (kD * albedo / PI + specular) * uLightColor * NdotL;

    float up = 0.5 + 0.5 * N.y;
    vec3 ambient = albedo * (0.18 + 0.12 * up);
    vec3 color = ambient + Lo;

    // Атмосфера: мягкое свечение по fresnel-краю
    if (uAlphaMode == 2) {
        float fresnel = pow(1.0 - max(dot(N, V), 0.0), 2.5);
        color = albedo * (0.6 + 1.4 * fresnel);
        alpha = clamp(alpha * (0.35 + 0.9 * fresnel), 0.0, 1.0);
    }

    FragColor = vec4(color, alpha);
}
