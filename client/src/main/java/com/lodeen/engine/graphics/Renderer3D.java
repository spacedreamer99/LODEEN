package com.lodeen.engine.graphics;

import com.lodeen.engine.scene.Camera;
import com.lodeen.engine.scene.GameObject;
import org.joml.*;
import java.util.List;
import static org.lwjgl.opengl.GL30.*;

public class Renderer3D {
    private final ShaderProgram shader;
    private final Vector3f lightPos = new Vector3f(5f, 5f, 5f);
    private final Vector3f lightColor = new Vector3f(3.5f, 3.5f, 3.5f);

    public Renderer3D() {
        shader = ShaderProgram.fromResources("/shaders/pbr.vert", "/shaders/pbr.frag");
    }

    public void render(List<GameObject> objects, Camera camera, int w, int h) {
        shader.use();
        setMat4("uView",  camera.getView());
        setMat4("uProj",  camera.getProjection(w, h));
        setVec3("uLightPos",   lightPos);
        setVec3("uLightColor", lightColor);
        setVec3("uCamPos", camera.getPosition());

        // Найти первую атмосферу (BLEND-материал) и проверить: камера внутри неё?
        Vector3f camPos = camera.getPosition();
        int camInside = 0;
        for (GameObject go : objects) {
            if (go.mesh == null || go.mesh.material == null) continue;
            if (go.mesh.material.alphaMode != 2) continue;
            Vector3f center = go.worldPosition();
            float r = go.boundingRadius();
            if (camPos.distance(center) < r) { camInside = 1; break; }
        }
        shader.setInt("uCamInside", camInside);

        for (GameObject go : objects) {
            if (go.mesh == null) continue;
            Matrix4f world = go.worldMatrix();
            setMat4("uModel", world);
            setMat3("uNormalMat", new Matrix3f(world).invert().transpose());
            Material m = go.mesh.material;
            setVec4("uBaseColorFactor", m.baseColorFactor);
            shader.setFloat("uMetallic",    m.metallic);
            shader.setFloat("uRoughness",   m.roughness);
            shader.setInt  ("uAlphaMode",   m.alphaMode);
            shader.setFloat("uAlphaCutoff", m.alphaCutoff);
            bind(m.baseColorTexture,         "uBaseColorTex", "uHasTexture", 0);
            bind(m.metallicRoughnessTexture, "uMRTex",        "uHasMRTex",   1);
            go.mesh.draw();
        }
    }

    private void bind(Texture tex, String sampler, String flag, int unit) {
        if (tex != null) {
            tex.bind(unit);
            shader.setInt(sampler, unit);
            shader.setInt(flag, 1);
        } else shader.setInt(flag, 0);
    }

    private void setMat4(String n, Matrix4f m) {
        try (var mem = org.lwjgl.system.MemoryStack.stackPush()) {
            glUniformMatrix4fv(shader.getUniformLocation(n), false, m.get(mem.mallocFloat(16)));
        }
    }
    private void setMat3(String n, Matrix3f m) {
        try (var mem = org.lwjgl.system.MemoryStack.stackPush()) {
            glUniformMatrix3fv(shader.getUniformLocation(n), false, m.get(mem.mallocFloat(9)));
        }
    }
    private void setVec3(String n, Vector3f v) {
        glUniform3f(shader.getUniformLocation(n), v.x, v.y, v.z);
    }
    private void setVec4(String n, Vector4f v) {
        glUniform4f(shader.getUniformLocation(n), v.x, v.y, v.z, v.w);
    }

    public void cleanup() { shader.cleanup(); }
}
