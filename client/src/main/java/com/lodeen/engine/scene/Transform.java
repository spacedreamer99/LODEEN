package com.lodeen.engine.scene;

import org.joml.Matrix4f;
import org.joml.Quaternionf;
import org.joml.Vector3f;

public class Transform {
    public final Vector3f position = new Vector3f();
    public final Quaternionf rotation = new Quaternionf();
    public final Vector3f scale = new Vector3f(1, 1, 1);

    public Matrix4f localMatrix() {
        return new Matrix4f()
            .translate(position)
            .rotate(rotation)
            .scale(scale);
    }

    /** @deprecated кеш убран — метод оставлен как no-op для совместимости. */
    @Deprecated
    public void markDirty() { /* no-op */ }
}
