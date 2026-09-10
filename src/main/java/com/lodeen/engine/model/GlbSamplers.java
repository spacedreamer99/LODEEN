package com.lodeen.engine.model;

import java.util.*;
import static org.lwjgl.opengl.GL30.*;

public class GlbSamplers {
    private final GlbLoader glb;
    public GlbSamplers(GlbLoader glb) { this.glb = glb; }

    public int magFilter(int texIdx) { return filter(texIdx, "magFilter", GL_LINEAR); }
    public int minFilter(int texIdx) { return filter(texIdx, "minFilter", GL_LINEAR_MIPMAP_LINEAR); }
    public int wrapS(int texIdx) { return wrap(texIdx, "wrapS"); }
    public int wrapT(int texIdx) { return wrap(texIdx, "wrapT"); }

    @SuppressWarnings("unchecked")
    private Map<String,Object> sampler(int texIdx) {
        List<Map<String,Object>> textures = (List<Map<String,Object>>) glb.json.get("textures");
        Map<String,Object> t = textures.get(texIdx);
        if (!t.containsKey("sampler")) return null;
        int sIdx = iv(t.get("sampler"));
        List<Map<String,Object>> samplers = (List<Map<String,Object>>) glb.json.get("samplers");
        if (samplers == null || sIdx >= samplers.size()) return null;
        return samplers.get(sIdx);
    }

    private int filter(int texIdx, String key, int fallback) {
        Map<String,Object> s = sampler(texIdx);
        if (s == null || !s.containsKey(key)) return fallback;
        switch (iv(s.get(key))) {
            case 9728: return GL_NEAREST;
            case 9729: return GL_LINEAR;
            case 9984: return GL_NEAREST_MIPMAP_NEAREST;
            case 9985: return GL_LINEAR_MIPMAP_NEAREST;
            case 9986: return GL_NEAREST_MIPMAP_LINEAR;
            case 9987: return GL_LINEAR_MIPMAP_LINEAR;
            default:   return fallback;
        }
    }

    private int wrap(int texIdx, String key) {
        Map<String,Object> s = sampler(texIdx);
        if (s == null || !s.containsKey(key)) return GL_REPEAT;
        switch (iv(s.get(key))) {
            case 33071: return GL_CLAMP_TO_EDGE;
            case 33648: return GL_MIRRORED_REPEAT;
            case 10497: return GL_REPEAT;
            default:    return GL_REPEAT;
        }
    }

    private static int iv(Object o) { return ((Number) o).intValue(); }
}
