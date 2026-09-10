package com.lodeen.engine.model;

import com.lodeen.engine.graphics.Mesh;
import org.joml.Matrix4f;

public class ObjectInstance {
    public Mesh mesh;
    public Matrix4f transform = new Matrix4f();
}
