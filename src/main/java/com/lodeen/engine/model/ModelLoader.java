package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Material;
import com.lodeen.engine.graphics.Mesh;
import com.lodeen.engine.graphics.Texture;
import com.lodeen.engine.scene.GameObject;
import de.javagl.jgltf.model.*;
import de.javagl.jgltf.model.io.GltfModelReader;
import de.javagl.jgltf.model.v2.MaterialModelV2;
import org.joml.*;

import java.io.InputStream;
import java.nio.ByteBuffer;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.util.ArrayList;
import java.util.IdentityHashMap;
import java.util.List;
import java.util.Map;

import static org.lwjgl.opengl.GL30.*;

public class ModelLoader {

    private static final Map<TextureModel, Texture> textureCache = new IdentityHashMap<>();

    public static List<GameObject> load(String resourcePath) {
        textureCache.clear();
        try {
            InputStream is = ModelLoader.class.getResourceAsStream(resourcePath);
            if (is == null) throw new RuntimeException("Resource not found: " + resourcePath);
            Path tempFile = Files.createTempFile("lodeen_", ".glb");
            Files.copy(is, tempFile, StandardCopyOption.REPLACE_EXISTING);

            GltfModelReader reader = new GltfModelReader();
            GltfModel model = reader.read(tempFile);
            Files.deleteIfExists(tempFile);

            List<GameObject> out = new ArrayList<>();
            for (NodeModel node : model.getNodeModels()) {
                if (node.getParent() == null) traverse(node, null, out);
            }
            return out;
        } catch (Exception e) {
            throw new RuntimeException("Failed to load GLB: " + resourcePath, e);
        }
    }

    private static void traverse(NodeModel node, GameObject parent, List<GameObject> out) {
        GameObject go = new GameObject();
        if (node.getName() != null) go.name = node.getName();

        float[] matrix = node.computeGlobalTransform(null);
        if (matrix != null) {
            Matrix4f mat = new Matrix4f().set(matrix);
            go.transform.position.set(mat.getTranslation(new Vector3f()));
            go.transform.rotation.set(mat.getUnnormalizedRotation(new Quaternionf()));
            go.transform.scale.set(mat.getScale(new Vector3f()));
        }

        if (parent != null) parent.addChild(go);
        out.add(go);

        for (MeshModel meshModel : node.getMeshModels()) {
            List<MeshPrimitiveModel> primitives = meshModel.getMeshPrimitiveModels();
            for (int i = 0; i < primitives.size(); i++) {
                MeshPrimitiveModel prim = primitives.get(i);
                Mesh mesh = buildMesh(prim);
                if (mesh == null) continue;

                if (prim.getMaterialModel() != null
                        && prim.getMaterialModel() instanceof MaterialModelV2) {
                    mesh.material = loadMaterial((MaterialModelV2) prim.getMaterialModel());
                }

                if (primitives.size() == 1) {
                    go.mesh = mesh;
                } else {
                    GameObject child = new GameObject();
                    child.name = go.name + "_prim" + i;
                    child.mesh = mesh;
                    go.addChild(child);
                    out.add(child);
                }
            }
        }

        for (NodeModel child : node.getChildren()) {
            traverse(child, go, out);
        }
    }

    private static Material loadMaterial(MaterialModelV2 src) {
        Material m = new Material();

        float[] bcf = src.getBaseColorFactor();
        if (bcf != null && bcf.length == 4) {
            m.baseColorFactor.set(bcf[0], bcf[1], bcf[2], bcf[3]);
        }
        m.metallic = src.getMetallicFactor();
        m.roughness = src.getRoughnessFactor();

        MaterialModelV2.AlphaMode am = src.getAlphaMode();
        if (am == MaterialModelV2.AlphaMode.BLEND) m.alphaMode = 2;
        else if (am == MaterialModelV2.AlphaMode.MASK) m.alphaMode = 1;
        else m.alphaMode = 0;
        m.alphaCutoff = src.getAlphaCutoff();

        if (src.getBaseColorTexture() != null) {
            m.baseColorTexture = loadTexture(src.getBaseColorTexture());
        }
        if (src.getMetallicRoughnessTexture() != null) {
            m.metallicRoughnessTexture = loadTexture(src.getMetallicRoughnessTexture());
        }
        return m;
    }

    private static Texture loadTexture(TextureModel tm) {
        return textureCache.computeIfAbsent(tm, t -> {
            ImageModel im = t.getImageModel();
            ByteBuffer bb = im.getImageData();
            byte[] bytes = new byte[bb.remaining()];
            bb.get(bytes);

            int mag  = mapFilter(t.getMagFilter(), GL_LINEAR);
            int min  = mapFilter(t.getMinFilter(), GL_LINEAR_MIPMAP_LINEAR);
            int wrapS = mapWrap(t.getWrapS(), GL_REPEAT);
            int wrapT = mapWrap(t.getWrapT(), GL_REPEAT);
            return new Texture(bytes, mag, min, wrapS, wrapT);
        });
    }

    private static int mapFilter(Integer f, int fallback) {
        if (f == null) return fallback;
        switch (f) {
            case 9728: return GL_NEAREST;
            case 9729: return GL_LINEAR;
            case 9984: return GL_NEAREST_MIPMAP_NEAREST;
            case 9985: return GL_LINEAR_MIPMAP_NEAREST;
            case 9986: return GL_NEAREST_MIPMAP_LINEAR;
            case 9987: return GL_LINEAR_MIPMAP_LINEAR;
            default:   return fallback;
        }
    }

    private static int mapWrap(Integer w, int fallback) {
        if (w == null) return fallback;
        switch (w) {
            case 33071: return GL_CLAMP_TO_EDGE;
            case 33648: return GL_MIRRORED_REPEAT;
            case 10497: return GL_REPEAT;
            default:    return fallback;
        }
    }

    private static Mesh buildMesh(MeshPrimitiveModel prim) {
        AccessorModel posAcc = prim.getAttributes().get("POSITION");
        if (posAcc == null) return null;
        float[] pos = readFloats(posAcc);

        AccessorModel norAcc = prim.getAttributes().get("NORMAL");
        float[] nor = (norAcc != null) ? readFloats(norAcc) : null;

        AccessorModel uvAcc = prim.getAttributes().get("TEXCOORD_0");
        float[] uv = (uvAcc != null) ? readFloats(uvAcc) : null;

        AccessorModel idxAcc = prim.getIndices();
        int[] indices = (idxAcc != null) ? readInts(idxAcc) : identity(pos.length / 3);

        int vertexCount = pos.length / 3;
        float[] verts = new float[vertexCount * 8];
        for (int i = 0; i < vertexCount; i++) {
            int o = i * 8, p = i * 3;
            verts[o] = pos[p];
            verts[o + 1] = pos[p + 1];
            verts[o + 2] = pos[p + 2];
            if (nor != null) {
                verts[o + 3] = nor[p];
                verts[o + 4] = nor[p + 1];
                verts[o + 5] = nor[p + 2];
            } else {
                verts[o + 4] = 1f;
            }
            if (uv != null) {
                verts[o + 6] = uv[i * 2];
                verts[o + 7] = uv[i * 2 + 1];
            }
        }
        return new Mesh(verts, indices, new int[]{3, 3, 2});
    }

    private static float[] readFloats(AccessorModel acc) {
        AccessorFloatData data = (AccessorFloatData) acc.getAccessorData();
        int count = acc.getCount();
        int comps = acc.getElementType().getNumComponents();
        float[] out = new float[count * comps];
        for (int i = 0; i < count; i++)
            for (int c = 0; c < comps; c++)
                out[i * comps + c] = data.get(i, c);
        return out;
    }

    private static int[] readInts(AccessorModel acc) {
        AccessorData raw = acc.getAccessorData();
        int count = acc.getCount();
        int[] out = new int[count];
        if (raw instanceof AccessorIntData) {
            AccessorIntData d = (AccessorIntData) raw;
            for (int i = 0; i < count; i++) out[i] = d.get(i, 0);
        } else if (raw instanceof AccessorShortData) {
            AccessorShortData d = (AccessorShortData) raw;
            for (int i = 0; i < count; i++) out[i] = d.get(i, 0) & 0xFFFF;
        } else if (raw instanceof AccessorByteData) {
            AccessorByteData d = (AccessorByteData) raw;
            for (int i = 0; i < count; i++) out[i] = d.get(i, 0) & 0xFF;
        } else {
            throw new RuntimeException("Unsupported index type: " + raw.getClass());
        }
        return out;
    }

    private static int[] identity(int n) {
        int[] a = new int[n];
        for (int i = 0; i < n; i++) a[i] = i;
        return a;
    }
}
