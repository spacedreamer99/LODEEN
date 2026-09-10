package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Material;
import com.lodeen.engine.graphics.Texture;
import java.nio.ByteBuffer;
import java.util.*;

public class GlbMaterials {
    private final GlbLoader glb;
    private final GlbSamplers samplers;
    private final Map<Integer, Material> matCache = new HashMap<>();
    private final Map<Integer, Texture> texCache = new HashMap<>();

    public GlbMaterials(GlbLoader glb) {
        this.glb = glb;
        this.samplers = new GlbSamplers(glb);
    }

    @SuppressWarnings("unchecked")
    public Material load(int idx) {
        if (idx < 0) return new Material();
        if (matCache.containsKey(idx)) return matCache.get(idx);
        List<Map<String,Object>> materials = (List<Map<String,Object>>) glb.json.get("materials");
        if (materials == null || idx >= materials.size()) return new Material();
        Map<String,Object> mat = materials.get(idx);
        Material m = new Material();
        if (mat.containsKey("name")) m.name = (String) mat.get("name");
        Map<String,Object> pbr = (Map<String,Object>) mat.get("pbrMetallicRoughness");
        if (pbr != null) readPbr(pbr, m);
        String mode = (String) mat.getOrDefault("alphaMode", "OPAQUE");
        m.alphaMode = mode.equals("BLEND") ? 2 : mode.equals("MASK") ? 1 : 0;
        if (mat.containsKey("alphaCutoff"))
            m.alphaCutoff = ((Number) mat.get("alphaCutoff")).floatValue();
        matCache.put(idx, m);
        return m;
    }

    @SuppressWarnings("unchecked")
    private void readPbr(Map<String,Object> pbr, Material m) {
        if (pbr.containsKey("baseColorFactor")) {
            List<Object> f = (List<Object>) pbr.get("baseColorFactor");
            m.baseColorFactor.set(fl(f,0), fl(f,1), fl(f,2), fl(f,3));
        }
        if (pbr.containsKey("metallicFactor"))
            m.metallic = ((Number) pbr.get("metallicFactor")).floatValue();
        if (pbr.containsKey("roughnessFactor"))
            m.roughness = ((Number) pbr.get("roughnessFactor")).floatValue();
        if (pbr.containsKey("baseColorTexture"))
            m.baseColorTexture = texture(iv(((Map<String,Object>) pbr.get("baseColorTexture")).get("index")));
        if (pbr.containsKey("metallicRoughnessTexture"))
            m.metallicRoughnessTexture = texture(iv(((Map<String,Object>) pbr.get("metallicRoughnessTexture")).get("index")));
    }

    private Texture texture(int texIdx) {
        return texCache.computeIfAbsent(texIdx, i -> new Texture(
            readImage(imageIndex(i)),
            samplers.magFilter(i), samplers.minFilter(i),
            samplers.wrapS(i), samplers.wrapT(i)));
    }

    @SuppressWarnings("unchecked")
    private int imageIndex(int texIdx) {
        List<Map<String,Object>> textures = (List<Map<String,Object>>) glb.json.get("textures");
        return iv(textures.get(texIdx).get("source"));
    }

    @SuppressWarnings("unchecked")
    private byte[] readImage(int imageIdx) {
        List<Map<String,Object>> images = (List<Map<String,Object>>) glb.json.get("images");
        int bvIdx = iv(images.get(imageIdx).get("bufferView"));
        Map<String,Object> bv = glb.bufferView(bvIdx);
        int off = bv.containsKey("byteOffset") ? iv(bv.get("byteOffset")) : 0;
        int len = iv(bv.get("byteLength"));
        ByteBuffer bin = glb.bin.duplicate();
        bin.position(off);
        byte[] out = new byte[len];
        bin.get(out);
        return out;
    }

    public void cleanup() {
        for (Texture t : texCache.values()) t.cleanup();
        texCache.clear(); matCache.clear();
    }

    private static int iv(Object o) { return ((Number) o).intValue(); }
    private static float fl(List<Object> l, int i) { return ((Number) l.get(i)).floatValue(); }
}
