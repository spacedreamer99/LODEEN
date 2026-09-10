package com.lodeen.engine.graphics;

import org.joml.Vector4f;

public class Material {
    public String name = "default";
    public Texture baseColorTexture;
    public Texture metallicRoughnessTexture;
    public Vector4f baseColorFactor = new Vector4f(1, 1, 1, 1);
    public float metallic = 1f;
    public float roughness = 1f;
    public int alphaMode = 0;
    public float alphaCutoff = 0.5f;
}
