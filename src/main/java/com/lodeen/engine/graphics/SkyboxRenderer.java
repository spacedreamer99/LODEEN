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
            glUniformMatrix3fv(shader.getUniformLocation("uCamRot"), false,
                rot.get(mem.mallocFloat(9)));
        }
        Vector3f cam = camera.getPosition();
        glUniform3f(shader.getUniformLocation("uCamPos"), cam.x, cam.y, cam.z);
        glUniform1f(shader.getUniformLocation("uFov"),
            (float) java.lang.Math.toRadians(camera.getFov()));
        glUniform2f(shader.getUniformLocation("uResolution"), width, height);
        glUniform1f(shader.getUniformLocation("uWaterRadius"), waterRadius);

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
