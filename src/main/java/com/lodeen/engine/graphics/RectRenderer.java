package com.lodeen.engine.graphics;

import static org.lwjgl.opengl.GL30.*;

public class RectRenderer {
    private ShaderProgram shader;
    private int vao, vbo;

    public RectRenderer() {
        shader = new ShaderProgram(
            "#version 330 core\n" +
            "layout (location=0) in vec2 aPos;\n" +
            "uniform vec4 uColor;\n" +
            "out vec4 vColor;\n" +
            "void main(){ gl_Position = vec4(aPos, 0.0, 1.0); vColor = uColor; }",
            "#version 330 core\n" +
            "in vec4 vColor;\n" +
            "out vec4 FragColor;\n" +
            "void main(){ FragColor = vColor; }"
        );
        vao = glGenVertexArrays();
        vbo = glGenBuffers();
        glBindVertexArray(vao);
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glVertexAttribPointer(0, 2, GL_FLOAT, false, 0, 0);
        glEnableVertexAttribArray(0);
    }

    public void draw(float x, float y, float w, float h, int screenW, int screenH,
                     float r, float g, float b, float a) {
        float ndcX = (x / screenW) * 2 - 1;
        float ndcY = 1 - (y / screenH) * 2;
        float ndcW = (w / screenW) * 2;
        float ndcH = (h / screenH) * 2;

        float[] verts = {
            ndcX, ndcY,
            ndcX + ndcW, ndcY,
            ndcX, ndcY - ndcH,
            ndcX + ndcW, ndcY - ndcH
        };
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glBufferData(GL_ARRAY_BUFFER, verts, GL_DYNAMIC_DRAW);
        shader.use();
        glUniform4f(glGetUniformLocation(shader.getId(), "uColor"), r, g, b, a);
        glBindVertexArray(vao);
        glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
    }

    public void cleanup() {
        glDeleteBuffers(vbo);
        glDeleteVertexArrays(vao);
        shader.cleanup();
    }
}