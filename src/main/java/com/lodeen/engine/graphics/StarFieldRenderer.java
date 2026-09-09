package com.lodeen.engine.graphics;

import static org.lwjgl.opengl.GL30.*;

public class StarFieldRenderer {
    private ShaderProgram shader;
    private int vao, vbo;

    public StarFieldRenderer() {
        shader = new ShaderProgram(
            "#version 330 core\n" +
            "layout (location=0) in vec2 aPos;\n" +
            "void main(){ gl_Position = vec4(aPos, 0.0, 1.0); }",
            "#version 330 core\n" +
            "out vec4 FragColor;\n" +
            "uniform vec2 resolution;\n" +
            "float hash(vec2 p) {\n" +
            "    return fract(sin(dot(p, vec2(12.9898,78.233))) * 43758.5453);\n" +
            "}\n" +
            "void main(){\n" +
            "    // Сохраняем пропорции: используем resolution.y как базовую единицу\n" +
            "    vec2 uv = gl_FragCoord.xy / resolution.y;\n" +
            "    float gridSize = 400.0;\n" +          // больше ячеек -> больше звёзд
            "    vec2 grid = floor(uv * gridSize);\n" +
            "    float star = step(0.999, hash(grid));\n" +
            "    FragColor = vec4(vec3(star), 1.0);\n" +
            "}"
        );
        float[] verts = {-1f,-1f, 1f,-1f, -1f,1f, 1f,1f};
        vao = glGenVertexArrays();
        vbo = glGenBuffers();
        glBindVertexArray(vao);
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glBufferData(GL_ARRAY_BUFFER, verts, GL_STATIC_DRAW);
        glVertexAttribPointer(0, 2, GL_FLOAT, false, 0, 0);
        glEnableVertexAttribArray(0);
    }

    public void render(int width, int height) {
        shader.use();
        glUniform2f(glGetUniformLocation(shader.getId(), "resolution"), width, height);
        glBindVertexArray(vao);
        glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
    }

    public void cleanup() {
        glDeleteBuffers(vbo);
        glDeleteVertexArrays(vao);
        shader.cleanup();
    }
}
