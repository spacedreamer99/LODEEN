package com.lodeen.engine.model;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.util.Map;

/**
 * Обёртка над Jackson ObjectMapper для совместимости с существующим API.
 * Позволяет постепенно мигрировать без переписывания GlbLoader.
 */
public class Json {
    private static final ObjectMapper MAPPER = new ObjectMapper();

    @SuppressWarnings("unchecked")
    public static Object parse(String src) {
        try {
            return MAPPER.readValue(src, Map.class);
        } catch (Exception e) {
            throw new RuntimeException("JSON parse failed", e);
        }
    }
}
