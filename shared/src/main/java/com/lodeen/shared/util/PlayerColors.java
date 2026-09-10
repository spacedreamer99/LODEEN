package com.lodeen.shared.util;

/** Детерминированный цвет игрока по его ID — уникальный для каждого игрока. */
public class PlayerColors {

    /** Возвращает RGB [0..1] по ID. Один и тот же ID всегда даёт один цвет. */
    public static float[] rgbOf(String id) {
        int hash = (id == null) ? 0 : id.hashCode();
        float hue = Math.abs(hash % 360);
        return hsvToRgb(hue, 0.7f, 0.95f);
    }

    private static float[] hsvToRgb(float h, float s, float v) {
        int i = (int) (h / 60f) % 6;
        float f = h / 60f - (int) (h / 60f);
        float p = v * (1 - s);
        float q = v * (1 - s * f);
        float t = v * (1 - s * (1 - f));
        return switch (i) {
            case 0 -> new float[]{v, t, p};
            case 1 -> new float[]{q, v, p};
            case 2 -> new float[]{p, v, t};
            case 3 -> new float[]{p, q, v};
            case 4 -> new float[]{t, p, v};
            default -> new float[]{v, p, q};
        };
    }
}
