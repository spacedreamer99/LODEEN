package com.lodeen.engine.scene;

import com.lodeen.engine.graphics.Mesh;
import org.joml.Matrix4f;
import org.joml.Vector3f;
import java.util.ArrayList;
import java.util.List;

public class GameObject {
    public String name = "";
    public final Transform transform = new Transform();
    public Mesh mesh;
    public GameObject parent;
    public final List<GameObject> children = new ArrayList<>();

    public void addChild(GameObject child) {
        child.parent = this;
        children.add(child);
    }

    public Matrix4f worldMatrix() {
        if (parent == null) return transform.localMatrix();
        return new Matrix4f(parent.worldMatrix()).mul(transform.localMatrix());
    }

    public Vector3f worldPosition() {
        return worldMatrix().getTranslation(new Vector3f());
    }

    public boolean isTransparent() {
        return mesh != null && mesh.material != null && mesh.material.alphaMode == 2;
    }

    public boolean isWater() {
        return mesh != null && mesh.material != null && mesh.material.roughness < 0.3f;
    }

    public float boundingRadius() {
        if (mesh == null) return 0f;
        float r = mesh.getBoundingRadius();
        float s = transform.scale.x;
        if (transform.scale.y > s) s = transform.scale.y;
        if (transform.scale.z > s) s = transform.scale.z;
        return r * s;
    }
}
