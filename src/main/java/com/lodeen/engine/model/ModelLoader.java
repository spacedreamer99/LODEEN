package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Mesh;
import org.joml.*;
import java.io.InputStream;
import java.util.*;

public class ModelLoader {

    @SuppressWarnings("unchecked")
    public static List<ObjectInstance> load(String path) {
        GlbLoader glb = new GlbLoader(readResource(path));
        GlbMaterials ml = new GlbMaterials(glb);
        List<ObjectInstance> out = new ArrayList<>();
        List<Map<String,Object>> scenes = (List<Map<String,Object>>) glb.json.get("scenes");
        List<Map<String,Object>> nodes  = (List<Map<String,Object>>) glb.json.get("nodes");
        List<Object> roots = (List<Object>) scenes.get(0).get("nodes");
        System.out.println("Scene roots: " + roots.size());
        for (Object ri : roots)
            traverse(glb, ml, nodes, iv(ri), new Matrix4f(), out);
        return out;
    }

    @SuppressWarnings("unchecked")
    private static void traverse(GlbLoader glb, GlbMaterials ml,
                                  List<Map<String,Object>> nodes, int idx,
                                  Matrix4f parent, List<ObjectInstance> out) {
        Map<String,Object> n = nodes.get(idx);
        Matrix4f local = new Matrix4f();
        if (n.containsKey("matrix")) {
            List<Object> m = (List<Object>) n.get("matrix");
            float[] a = new float[16];
            for (int i = 0; i < 16; i++) a[i] = ((Number) m.get(i)).floatValue();
            local.set(a);
        } else {
            if (n.containsKey("translation")) {
                List<Object> t = (List<Object>) n.get("translation");
                local.translate(fl(t,0), fl(t,1), fl(t,2));
            }
            if (n.containsKey("rotation")) {
                List<Object> r = (List<Object>) n.get("rotation");
                local.rotate(new Quaternionf(fl(r,0), fl(r,1), fl(r,2), fl(r,3)));
            }
            if (n.containsKey("scale")) {
                List<Object> s = (List<Object>) n.get("scale");
                local.scale(fl(s,0), fl(s,1), fl(s,2));
            }
        }
        Matrix4f world = new Matrix4f(parent).mul(local);
        System.out.println("Node[" + idx + "] mesh=" + n.get("mesh")
            + " scale=" + new Vector3f(local.getScale(new Vector3f())));

        if (n.containsKey("mesh")) {
            int meshIdx = iv(n.get("mesh"));
            List<Map<String,Object>> meshes = (List<Map<String,Object>>) glb.json.get("meshes");
            List<Map<String,Object>> prims = (List<Map<String,Object>>) meshes.get(meshIdx).get("primitives");
            for (Map<String,Object> prim : prims) {
                Mesh m = build(glb, prim);
                int matIdx = prim.containsKey("material") ? iv(prim.get("material")) : -1;
                m.material = ml.load(matIdx);
                ObjectInstance oi = new ObjectInstance();
                oi.mesh = m;
                oi.transform = new Matrix4f(world);
                out.add(oi);
            }
        }
        if (n.containsKey("children")) {
            List<Object> ch = (List<Object>) n.get("children");
            for (Object c : ch) traverse(glb, ml, nodes, iv(c), world, out);
        }
    }

    @SuppressWarnings("unchecked")
    private static Mesh build(GlbLoader glb, Map<String,Object> prim) {
        Map<String,Object> attrs = (Map<String,Object>) prim.get("attributes");
        int posIdx = iv(attrs.get("POSITION"));
        int norIdx = attrs.containsKey("NORMAL") ? iv(attrs.get("NORMAL")) : -1;
        int uvIdx  = attrs.containsKey("TEXCOORD_0") ? iv(attrs.get("TEXCOORD_0")) : -1;

        float[] pos = glb.readFloats(posIdx);
        float[] nor = norIdx >= 0 ? glb.readFloats(norIdx) : null;
        float[] uv  = uvIdx  >= 0 ? glb.readFloats(uvIdx)  : null;

        int n = pos.length / 3;
        int[] indices = prim.containsKey("indices")
            ? glb.readInts(iv(prim.get("indices"))) : identity(n);

        float[] verts = new float[n * 8];
        for (int i = 0; i < n; i++) {
            verts[i*8]     = pos[i*3];
            verts[i*8 + 1] = pos[i*3 + 1];
            verts[i*8 + 2] = pos[i*3 + 2];
            if (nor != null) {
                verts[i*8 + 3] = nor[i*3];
                verts[i*8 + 4] = nor[i*3 + 1];
                verts[i*8 + 5] = nor[i*3 + 2];
            } else verts[i*8 + 4] = 1f;
            if (uv != null) {
                verts[i*8 + 6] = uv[i*2];
                verts[i*8 + 7] = uv[i*2 + 1];
            }
        }
        return new Mesh(verts, indices, new int[]{3, 3, 2});
    }

    private static int[] identity(int n) { int[] a = new int[n]; for (int i=0;i<n;i++) a[i]=i; return a; }
    private static int iv(Object o) { return ((Number) o).intValue(); }
    private static float fl(List<Object> l, int i) { return ((Number) l.get(i)).floatValue(); }

    private static byte[] readResource(String path) {
        try (InputStream is = ModelLoader.class.getResourceAsStream(path)) {
            if (is == null) throw new RuntimeException("Resource not found: " + path);
            return is.readAllBytes();
        } catch (Exception e) { throw new RuntimeException(e); }
    }
}
