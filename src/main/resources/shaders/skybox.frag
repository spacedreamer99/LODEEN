#version 330 core
in vec2 vNDC;
out vec4 FragColor;

uniform mat3 uCamRot;
uniform float uFov;
uniform vec2 uResolution;
uniform int uUnderwater;
uniform float uUnderwaterDensity;

// #include "sky.glsl"

void main() {
    float aspect = uResolution.x / uResolution.y;
    float focal = 1.0 / tan(uFov * 0.5);
    vec3 viewDir = normalize(vec3(vNDC.x * aspect, vNDC.y, -focal));
    vec3 dir = uCamRot * viewDir;

    vec3 color;

    if (uUnderwater == 1) {
        // Под водой: свет проникает СВЕРХУ через поверхность.
        // Смотрим вверх — яркое небо; к горизонту — темнее; вниз — почти чёрное дно.
        float upness = clamp(dir.y, -1.0, 1.0);

        // Цвет в зените — светлый синий-голубой (как солнце через воду)
        vec3 zenithCol  = vec3(0.55, 0.75, 0.95);
        // У горизонта — средний синий
        vec3 horizonCol = vec3(0.06, 0.18, 0.38);
        // Дно — почти чёрный синий
        vec3 bottomCol  = vec3(0.005, 0.02, 0.07);

        // Плавный переход: up > 0 — от горизонта к зениту, up < 0 — от горизонта к дну
        float tUp   = pow(max(upness, 0.0), 0.6);
        float tDown = pow(max(-upness, 0.0), 0.6);

        color = mix(horizonCol, zenithCol, tUp);
        color = mix(color, bottomCol, tDown);
    } else {
        color = skySample(dir);
    }

    FragColor = vec4(color, 1.0);
}
