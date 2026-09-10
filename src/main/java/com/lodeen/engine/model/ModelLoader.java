package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Mesh;
import com.lodeen.engine.scene.GameObject;
import org.joml.*;
import java.io.InputStream;
import java.util.*;

public class ModelLoader {
    @SuppressWarnings("unchecked")
    public static List<GameObject> load(String path) {
        GlbLoader glb = new GlbLoader(readResource(path));
        GlbMaterials ml = new GlbMaterials(glb);
        List<GameObject> out = new ArrayList<>();
        List<Map<String,Object>> scenes = (List<Map<String,Object>>) glb.json.get("scenes");
        List<Map<String,Object>> nodes  = (List<Map<String,Object>>) glb.json.get("nodes");
        List<Object> roots = (List<Object>) scenes.get(0).get("nodes");
        for (Object ri : roots) traverse(glb, ml, nodes, iv(ri), null, out);
        return out;
    }

    @SuppressWarnings("unchecked")
    private static void traverse(GlbLoader glb, GlbMaterials ml,
                                  List<Map<String,Object>> nodes, int idx,
                                  GameObject parent, List<GameObject> out) {
        Map<String,Object> n = nodes.get(idx);
        GameObject go = new GameObject();
        if (n.containsKey("name")) go.name = (String) n.get("name");
        if (n.containsKey("matrix")) {
            List<Object> m = (List<Object>) n.get("matrix");
            float[] a = new float[16];
            for (int i = 0; i < 16; i++) a[i] = ((Number) m.get(i)).floatValue();
            Matrix4f mat = new Matrix4f().set(a);
            go.transform.position.set(mat.getTranslation(new Vector3f()));
            go.transform.rotation.set(mat.getUnnormalizedRotation(new Quaternionf()));
            go.transform.scale.set(mat.getScale(new Vector3f()));
        } else {
            if (n.containsKey("translation")) {
                List<Object> t = (List<Object>) n.get("translation");
                go.transform.position.set(fl(t,0), fl(t,1), fl(t,2));
            }
            if (n.containsKey("rotation")) {
                List<Object> r = (List<Object>) n.get("rotation");
                go.transform.rotation.set(fl(r,0), fl(r,1), fl(r,2), fl(r,3));
            }
            if (n.containsKey("scale")) {
                List<Object> s = (List<Object>) n.get("scale");
                go.transform.scale.set(fl(s,0), fl(s,1), fl(s,2));
            }
        }
        go.transform.markDirty();
        if (parent != null) parent.addChild(go);
        out.add(go);
        if (n.containsKey("mesh")) {
            List<Map<String,Object>> meshes = (List<Map<String,Object>>) glb.json.get("meshes");
            List<Map<String,Object>> prims = (List<Map<String,Object>>) meshes.get(iv(n.get("mesh"))).get("primitives");
            for (Map<String,Object> prim : prims) {
                Mesh m = build(glb, prim);
                m.material = ml.load(prim.containsKey("material") ? iv(prim.get("material")) : -1);
                go.mesh = m;
            }
        }
        if (n.containsKey("children"))
            for (Object c : (List<Object>) n.get("children"))
                traverse(glb, ml, nodes, iv(c), go, out);
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
            int o = i * 8, p = i * 3;
            verts[o] = pos[p]; verts[o+1] = pos[p+1]; verts[o+2] = pos[p+2];
            if (nor != null) { verts[o+3] = nor[p]; verts[o+4] = nor[p+1]; verts[o+5] = nor[p+2]; }
            else verts[o+4] = 1f;
            if (uv != null) { verts[o+6] = uv[i*2]; verts[o+7] = uv[i*2+1]; }
        }
        return new Mesh(verts, indices, new int[]{3, 3, 2});
    }

    private static int[] identity(int n){int[] a=new int[n];for(int i=0;i<n;i++)a[i]=i;return a;}
    private static int iv(Object o){return((Number)o).intValue();}
    private static float fl(List<Object> l, int i) { return ((Number) l.get(i)).floatValue(); }
    private static byte[] readResource(String path) {
        try (InputStream is = ModelLoader.class.getResourceAsStream(path)) {
            if (is == null) throw new RuntimeException("Resource not found: " + path);
            return is.readAllBytes();
        } catch (Exception e) { throw new RuntimeException(e); }
    }
}
