package com.lodeen.engine.graphics;

import java.awt.*;
import java.awt.image.BufferedImage;
import java.nio.ByteBuffer;
import static org.lwjgl.opengl.GL30.*;

public class TextRenderer {
    private ShaderProgram shader;
    private int vao, vbo;

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

    // Новый метод: рисует текст так, чтобы его центр был в (centerX, centerY)
    public void drawTextCentered(String text, float centerX, float centerY, float scale,
                                 int screenW, int screenH) {
        TextureData data = createTexture(text);
        int tex = data.id;
        int texW = data.width;
        int texH = data.height;

        // Вычисляем левый верхний угол с учётом размеров текстуры
        float quadW = texW * scale;
        float quadH = texH * scale;
        float x = centerX - quadW / 2;
        float y = centerY - quadH / 2;

        float ndcX = (x / screenW) * 2 - 1;
        float ndcY = 1 - (y / screenH) * 2;
        float ndcW = (quadW / screenW) * 2;
        float ndcH = (quadH / screenH) * 2;

        shader.use();
        glUniform2f(glGetUniformLocation(shader.getId(), "uPos"), ndcX, ndcY - ndcH);
        glUniform2f(glGetUniformLocation(shader.getId(), "uScale"), ndcW, ndcH);
        glActiveTexture(GL_TEXTURE0);
        glBindTexture(GL_TEXTURE_2D, tex);
        shader.setInt("uTex", 0);
        glBindVertexArray(vao);
        glDrawArrays(GL_TRIANGLE_STRIP, 0, 4);
        glDeleteTextures(tex);
    }

    private TextureData createTexture(String text) {
        BufferedImage temp = new BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB);
        Graphics2D gTemp = temp.createGraphics();
        gTemp.setFont(new Font("Arial", Font.PLAIN, 24));
        FontMetrics fm = gTemp.getFontMetrics();
        int textWidth = fm.stringWidth(text);
        int textHeight = fm.getHeight();
        gTemp.dispose();

        int padding = 4;
        int imgW = textWidth + padding * 2;
        int imgH = textHeight + padding * 2;
        BufferedImage img = new BufferedImage(imgW, imgH, BufferedImage.TYPE_INT_ARGB);
        Graphics2D g = img.createGraphics();
        g.setRenderingHint(RenderingHints.KEY_ANTIALIASING, RenderingHints.VALUE_ANTIALIAS_ON);
        g.setFont(new Font("Arial", Font.PLAIN, 24));
        g.setColor(Color.WHITE);
        g.drawString(text, padding, fm.getAscent() + padding);
        g.dispose();

        int[] pixels = img.getRGB(0, 0, imgW, imgH, null, 0, imgW);
        ByteBuffer buf = ByteBuffer.allocateDirect(pixels.length * 4);
        for (int p : pixels) {
            buf.put((byte) ((p >> 16) & 0xFF));
            buf.put((byte) ((p >> 8) & 0xFF));
            buf.put((byte) (p & 0xFF));
            buf.put((byte) ((p >> 24) & 0xFF));
        }
        buf.flip();

        int tex = glGenTextures();
        glBindTexture(GL_TEXTURE_2D, tex);
        glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
        glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
        glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
        glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, imgW, imgH, 0, GL_RGBA, GL_UNSIGNED_BYTE, buf);

        return new TextureData(tex, imgW, imgH);
    }

    private static class TextureData {
        int id;
        int width;
        int height;
        TextureData(int id, int w, int h) {
            this.id = id;
            this.width = w;
            this.height = h;
        }
    }

    public void cleanup() {
        glDeleteBuffers(vbo);
        glDeleteVertexArrays(vao);
        shader.cleanup();
    }
}
