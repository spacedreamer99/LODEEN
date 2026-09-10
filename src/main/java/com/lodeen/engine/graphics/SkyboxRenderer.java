package com.lodeen.engine.graphics;

import com.lodeen.engine.scene.Camera;
import org.joml.Matrix3f;
import org.joml.Vector3f;
import static org.lwjgl.opengl.GL30.*;

public class SkyboxRenderer {
    private final ShaderProgram shader;
    private int vao, vbo;
    private float waterRadius = 1f;

    public SkyboxRenderer() {
        shader = ShaderProgram.fromResources("/shaders/skybox.vert", "/shaders/skybox.frag");
        float[] verts = { -1f, -1f, 1f, -1f, -1f, 1f, 1f, 1f };
        vao = glGenVertexArrays();
        vbo = glGenBuffers();
        glBindVertexArray(vao);
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glBufferData(GL_ARRAY_BUFFER, verts, GL_STATIC_DRAW);
        glVertexAttribPointer(0, 2, GL_FLOAT, false, 0, 0);
        glEnableVertexAttribArray(0);
        glBindVertexArray(0);
    }

    public void setWaterRadius(float r) { this.waterRadius = r; }

    public void render(Camera camera, int width, int height) {
        glDepthMask(false);
        glDisable(GL_DEPTH_TEST);
        glDisable(GL_CULL_FACE);

        shader.use();
        Matrix3f rot = new Matrix3f().rotation(camera.getRotation());
        try (var mem = org.lwjgl.system.MemoryStack.stackPush()) {
            glUniformMatrix3fv(glGetUniformLocation(shader.getId(), "uCamRot"), false,
                rot.get(mem.mallocFloat(9)));
        }
        glUniform1f(glGetUniformLocation(shader.getId(), "uFov"),
            (float) java.lang.Math.toRadians(camera.getFov()));
        glUniform2f(glGetUniformLocation(shader.getId(), "uResolution"), width, height);

        // Параметры подводного тумана — те же, что в Renderer3D
        Vector3f cam = camera.getPosition();
        float camDist = (float) java.lang.Math.sqrt(cam.x*cam.x + cam.y*cam.y + cam.z*cam.z);
        boolean underwater = camDist < waterRadius;
        shader.setInt("uUnderwater", underwater ? 1 : 0);
        float depth = java.lang.Math.max(0f, waterRadius - camDist);
        float density = 1.0f;
        shader.setFloat("uUnderwaterDensity", density);

        glBindVertexArray(vao);
        glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
        glBindVertexArray(0);

        glEnable(GL_DEPTH_TEST);
        glDepthMask(true);
    }

    public void cleanup() {
        glDeleteBuffers(vbo);
        glDeleteVertexArrays(vao);
        shader.cleanup();
    }
}
