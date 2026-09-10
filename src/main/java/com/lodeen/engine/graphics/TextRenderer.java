package com.lodeen.engine.graphics;

import java.util.HashMap;
import java.util.Map;
import static org.lwjgl.opengl.GL30.*;

public class TextRenderer {
    private ShaderProgram shader;
    private int vao, vbo;
    private final Map<String, FontTexture> cache = new HashMap<>();

    public TextRenderer() {
        shader = new ShaderProgram(
            "#version 330 core\n" +
            "layout (location=0) in vec2 aPos;\n" +
            "layout (location=1) in vec2 aTex;\n" +
            "out vec2 vTex;\n" +
            "uniform vec2 uPos;\n" +
            "uniform vec2 uScale;\n" +
            "void main(){ gl_Position = vec4(aPos * uScale + uPos, 0.0, 1.0); vTex = aTex; }",
            "#version 330 core\n" +
            "in vec2 vTex;\n" +
            "out vec4 FragColor;\n" +
            "uniform sampler2D uTex;\n" +
            "void main(){ FragColor = texture(uTex, vTex); }"
        );
        float[] verts = {
            0f, 0f, 0f, 1f,
            1f, 0f, 1f, 1f,
            0f, 1f, 0f, 0f,
            1f, 1f, 1f, 0f
        };
        vao = glGenVertexArrays();
        vbo = glGenBuffers();
        glBindVertexArray(vao);
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glBufferData(GL_ARRAY_BUFFER, verts, GL_STATIC_DRAW);
        int stride = 4 * Float.BYTES;
        glVertexAttribPointer(0, 2, GL_FLOAT, false, stride, 0);
        glVertexAttribPointer(1, 2, GL_FLOAT, false, stride, 2 * Float.BYTES);
        glEnableVertexAttribArray(0);
        glEnableVertexAttribArray(1);
    }

    public void drawTextCentered(String text, float centerX, float centerY, int pixelFontSize,
                                 int screenW, int screenH) {
        String key = pixelFontSize + "|" + text;
        FontTexture ft = cache.computeIfAbsent(key, k -> new FontTexture(text, pixelFontSize));

        float quadW = ft.width;
        float quadH = ft.height;
        float x = centerX - quadW / 2;
        float y = centerY - quadH / 2;

        float ndcX = (x / screenW) * 2 - 1;
        float ndcY = 1 - (y / screenH) * 2;
        float ndcW = (quadW / screenW) * 2;
        float ndcH = (quadH / screenH) * 2;

        shader.use();
        shader.setInt("uTex", 0);
        glUniform2f(shader.getUniformLocation("uPos"),   ndcX, ndcY - ndcH);
        glUniform2f(shader.getUniformLocation("uScale"), ndcW, ndcH);
        glActiveTexture(GL_TEXTURE0);
        glBindTexture(GL_TEXTURE_2D, ft.id);
        glBindVertexArray(vao);
        glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
    }

    public void cleanup() {
        for (FontTexture ft : cache.values()) ft.cleanup();
        cache.clear();
        glDeleteBuffers(vbo);
        glDeleteVertexArrays(vao);
        shader.cleanup();
    }
}
