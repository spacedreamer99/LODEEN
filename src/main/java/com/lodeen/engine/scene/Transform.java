package com.lodeen.engine.scene;

import org.joml.Matrix4f;
import org.joml.Quaternionf;
import org.joml.Vector3f;

public class Transform {
    public final Vector3f position = new Vector3f();
    public final Quaternionf rotation = new Quaternionf();
    public final Vector3f scale = new Vector3f(1, 1, 1);
    private final Matrix4f local = new Matrix4f();
    private boolean dirty = true;

    public void markDirty() { dirty = true; }

    public Matrix4f localMatrix() {
        if (dirty) {
            local.identity()
                .translate(position)
                .rotate(rotation)
                .scale(scale);
            dirty = false;
        }
        return local;
    }
}
