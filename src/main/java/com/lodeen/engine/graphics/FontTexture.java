package com.lodeen.engine.graphics;

import java.awt.*;
import java.awt.image.BufferedImage;
import java.io.InputStream;
import java.nio.ByteBuffer;
import static org.lwjgl.opengl.GL30.*;

public class FontTexture {
    public int id;
    public int width;
    public int height;

    public FontTexture(String text, int pixelFontSize) {
        Font font = loadFont().deriveFont((float) pixelFontSize);
        BufferedImage temp = new BufferedImage(1, 1, BufferedImage.TYPE_INT_ARGB);
        Graphics2D gTemp = temp.createGraphics();
        gTemp.setFont(font);
        FontMetrics fm = gTemp.getFontMetrics();
        int textWidth = fm.stringWidth(text);
        int textHeight = fm.getHeight();
        gTemp.dispose();

        int padX = Math.max(6, pixelFontSize / 3);
        int padY = Math.max(6, pixelFontSize / 3);
        width = textWidth + padX * 2;
        height = textHeight + padY * 2;

        BufferedImage img = new BufferedImage(width, height, BufferedImage.TYPE_INT_ARGB);
        Graphics2D g = img.createGraphics();
        g.setRenderingHint(RenderingHints.KEY_ANTIALIASING, RenderingHints.VALUE_ANTIALIAS_ON);
        g.setRenderingHint(RenderingHints.KEY_TEXT_ANTIALIASING, RenderingHints.VALUE_TEXT_ANTIALIAS_ON);
        g.setRenderingHint(RenderingHints.KEY_RENDERING, RenderingHints.VALUE_RENDER_QUALITY);
        g.setRenderingHint(RenderingHints.KEY_STROKE_CONTROL, RenderingHints.VALUE_STROKE_PURE);
        g.setFont(font);
        g.setColor(Color.WHITE);
        g.drawString(text, padX, padY + fm.getAscent());
        g.dispose();

        int[] pixels = img.getRGB(0, 0, width, height, null, 0, width);
        ByteBuffer buf = ByteBuffer.allocateDirect(pixels.length * 4);
        for (int p : pixels) {
            buf.put((byte) ((p >> 16) & 0xFF));
            buf.put((byte) ((p >> 8) & 0xFF));
            buf.put((byte) (p & 0xFF));
            buf.put((byte) ((p >> 24) & 0xFF));
        }
        buf.flip();

        id = glGenTextures();
        glBindTexture(GL_TEXTURE_2D, id);
        glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, GL_LINEAR);
        glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, GL_LINEAR);
        glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
        glTexImage2D(GL_TEXTURE_2D, 0, GL_RGBA, width, height, 0, GL_RGBA, GL_UNSIGNED_BYTE, buf);
    }

    private Font loadFont() {
        try (InputStream is = getClass().getResourceAsStream("/fonts/Forum-Regular.ttf")) {
            if (is == null) {
                System.err.println("Forum font not found, using default");
                return new Font("Serif", Font.PLAIN, 48);
            }
            return Font.createFont(Font.TRUETYPE_FONT, is);
        } catch (Exception e) {
            e.printStackTrace();
            return new Font("Serif", Font.PLAIN, 48);
        }
    }

    public void cleanup() {
        glDeleteTextures(id);
    }
}
