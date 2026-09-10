package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Material;
import com.lodeen.engine.graphics.Texture;
import java.nio.ByteBuffer;
import java.util.*;

public class GlbMaterials {
    private final GlbLoader glb;
    private final Map<Integer, Material> matCache = new HashMap<>();
    private final Map<Integer, Texture> texCache = new HashMap<>();

    public GlbMaterials(GlbLoader glb) { this.glb = glb; }

    @SuppressWarnings("unchecked")
    public Material load(int idx) {
        if (idx < 0) return new Material();
        if (matCache.containsKey(idx)) return matCache.get(idx);
        List<Map<String,Object>> materials =
            (List<Map<String,Object>>) glb.json.get("materials");
        if (materials == null || idx >= materials.size()) return new Material();
        Map<String,Object> mat = materials.get(idx);
        Material m = new Material();
        if (mat.containsKey("name")) m.name = (String) mat.get("name");

        Map<String,Object> pbr = (Map<String,Object>) mat.get("pbrMetallicRoughness");
        if (pbr != null) {
            if (pbr.containsKey("baseColorFactor")) {
                List<Object> f = (List<Object>) pbr.get("baseColorFactor");
                m.baseColorFactor.set(fl(f,0), fl(f,1), fl(f,2), fl(f,3));
            }
            if (pbr.containsKey("metallicFactor"))
                m.metallic = ((Number) pbr.get("metallicFactor")).floatValue();
            if (pbr.containsKey("roughnessFactor"))
                m.roughness = ((Number) pbr.get("roughnessFactor")).floatValue();
            if (pbr.containsKey("baseColorTexture")) {
                Map<String,Object> bct = (Map<String,Object>) pbr.get("baseColorTexture");
                m.baseColorTexture = texture(iv(bct.get("index")));
            }
        }
        String mode = (String) mat.getOrDefault("alphaMode", "OPAQUE");
        m.alphaMode = mode.equals("BLEND") ? 2 : mode.equals("MASK") ? 1 : 0;
        if (mat.containsKey("alphaCutoff"))
            m.alphaCutoff = ((Number) mat.get("alphaCutoff")).floatValue();

        System.out.println("Material[" + idx + "]: " + m.name
            + ", alphaMode=" + mode + ", hasTex=" + (m.baseColorTexture != null));
        matCache.put(idx, m);
        return m;
    }

    @SuppressWarnings("unchecked")
    private Texture texture(int texIdx) {
        return texCache.computeIfAbsent(texIdx, i -> {
            List<Map<String,Object>> textures =
                (List<Map<String,Object>>) glb.json.get("textures");
            Map<String,Object> t = textures.get(i);
            int src = iv(t.get("source"));
            return new Texture(readImage(src));
        });
    }

    @SuppressWarnings("unchecked")
    private byte[] readImage(int imageIdx) {
        List<Map<String,Object>> images =
            (List<Map<String,Object>>) glb.json.get("images");
        Map<String,Object> img = images.get(imageIdx);
        int bvIdx = iv(img.get("bufferView"));
        Map<String,Object> bv = glb.bufferView(bvIdx);
        int off = bv.containsKey("byteOffset") ? iv(bv.get("byteOffset")) : 0;
        int len = iv(bv.get("byteLength"));
        ByteBuffer bin = glb.bin.duplicate();
        byte[] out = new byte[len];
        bin.position(off);
        bin.get(out);
        System.out.println("Image[" + imageIdx + "]: " + len + " bytes");
        return out;
    }

    public Collection<Texture> allTextures() { return texCache.values(); }
    public void cleanup() {
        for (Texture t : texCache.values()) t.cleanup();
        texCache.clear(); matCache.clear();
    }

    private static int iv(Object o) { return ((Number) o).intValue(); }
    private static float fl(List<Object> l, int i) {
        return ((Number) l.get(i)).floatValue();
    }
}
