#version 330 core
in vec2 vNDC;
out vec4 FragColor;

uniform mat3 uCamRot;
uniform vec3 uCamPos;
uniform float uFov;
uniform vec2 uResolution;
uniform float uWaterRadius;

// #include "sky.glsl"

// Пересечение луча со сферой радиуса r в начале координат.
// Возвращает (tNear, tFar) или (-1,-1) если промах.
vec2 raySphere(vec3 orig, vec3 dir, float r) {
    float b = dot(orig, dir);
    float c = dot(orig, orig) - r * r;
    float disc = b * b - c;
    if (disc < 0.0) return vec2(-1.0);
    float sq = sqrt(disc);
    return vec2(-b - sq, -b + sq);
}

vec3 underwaterColor(vec3 dir) {
    float upness = clamp(dir.y, -1.0, 1.0);
    vec3 zenithCol  = vec3(0.55, 0.75, 0.95);
    vec3 horizonCol = vec3(0.06, 0.18, 0.38);
    vec3 bottomCol  = vec3(0.005, 0.02, 0.07);
    float tUp   = pow(max(upness, 0.0), 0.6);
    float tDown = pow(max(-upness, 0.0), 0.6);
    vec3 col = mix(horizonCol, zenithCol, tUp);
    col = mix(col, bottomCol, tDown);
    return col;
}

void main() {
    float aspect = uResolution.x / uResolution.y;
    float focal = 1.0 / tan(uFov * 0.5);
    vec3 viewDir = normalize(vec3(vNDC.x * aspect, vNDC.y, -focal));
    vec3 dir = uCamRot * viewDir;

    float camDistSq = dot(uCamPos, uCamPos);
    bool cameraUnderwater = camDistSq < uWaterRadius * uWaterRadius;

    vec3 color;
    vec2 hit = raySphere(uCamPos, dir, uWaterRadius);

    if (cameraUnderwater) {
        // Камера под водой
        if (hit.y > 0.0) {
            // Луч выходит из воды — небо сквозь воду
            vec3 sky = skySample(dir);
            float depthInWater = hit.y;
            float fogAmount = clamp(1.0 - exp(-depthInWater * 0.6), 0.0, 1.0);
            color = mix(sky, underwaterColor(dir), fogAmount);
        } else {
            color = underwaterColor(dir);
        }
    } else {
        // Камера над водой
        if (hit.x > 0.0) {
            // Луч попадает в воду — видим воду
            color = underwaterColor(dir);
        } else {
            // Луч в небо — звёзды
            color = skySample(dir);
        }
    }

    FragColor = vec4(color, 1.0);
}
