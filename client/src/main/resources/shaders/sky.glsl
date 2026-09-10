float hash13(vec3 p) {
    p = fract(p * 0.1031);
    p += dot(p, p.yzx + 33.33);
    return fract((p.x + p.y) * p.z);
}

vec3 hash33(vec3 p) {
    return vec3(hash13(p),
                hash13(p + vec3(13.1, 7.7, 3.3)),
                hash13(p + vec3(41.7, 23.9, 17.5)));
}

vec3 starColor(float r) {
    if (r > 0.48) return vec3(1.0, 0.6, 0.3);
    if (r > 0.28) return vec3(1.0, 0.8, 0.6);
    if (r > 0.16) return vec3(1.0, 0.95, 0.8);
    if (r > 0.08) return vec3(0.9, 0.9, 1.0);
    if (r > 0.03) return vec3(0.7, 0.8, 1.0);
    if (r > 0.01) return vec3(0.5, 0.6, 1.0);
    return vec3(0.3, 0.4, 1.0);
}

// Возвращает цвет звёздного неба для направления dir.
vec3 skySample(vec3 dir) {
    const float density = 90.0;
    vec3 p = dir * density;
    vec3 cell = floor(p);
    vec3 color = vec3(0.0);
    for (int dx = -1; dx <= 1; dx++)
    for (int dy = -1; dy <= 1; dy++)
    for (int dz = -1; dz <= 1; dz++) {
        vec3 nCell = cell + vec3(dx, dy, dz);
        vec3 nCenter = nCell + 0.5;
        vec3 diff = p - nCenter;
        if (dot(diff, diff) > 0.81) continue;
        if (hash13(nCell) < 0.6) continue;
        vec3 offsets = hash33(nCell);
        vec3 starPos = nCenter + (offsets - 0.5) * 0.7;
        float d = length(p - starPos);
        float sr = hash13(nCell + 23.4);
        float cr = hash13(nCell + 17.5);
        float size = 0.075 + 0.075 * sr;
        float x = d / size;
        if (x >= 1.0) continue;
        float b = 1.0 - x * x;
        b *= b;
        color += starColor(cr) * b;
    }
    return color;
}
