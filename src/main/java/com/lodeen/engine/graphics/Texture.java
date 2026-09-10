package com.lodeen.engine.graphics;

import javax.imageio.ImageIO;
import java.awt.image.BufferedImage;
import java.io.ByteArrayInputStream;
import java.nio.ByteBuffer;
import static org.lwjgl.opengl.GL30.*;

public class Texture {
    private final int id;
    private final int width, height;

    public Texture(byte[] imageBytes) {
        this(imageBytes, GL_LINEAR, GL_LINEAR_MIPMAP_LINEAR, GL_REPEAT, GL_REPEAT);
    }

    public Texture(byte[] imageBytes, int magFilter, int minFilter, int wrapS, int wrapT) {
        try {
            BufferedImage img = ImageIO.read(new ByteArrayInputStream(imageBytes));
            if (img == null) throw new RuntimeException("Unsupported image format");
            width = img.getWidth();
            height = img.getHeight();
            int[] px = img.getRGB(0, 0, width, height, null, 0, width);
            ByteBuffer buf = ByteBuffer.allocateDirect(px.length * 4);
            for (int p : px) {
                buf.put((byte) ((p >> 16) & 0xFF));
                buf.put((byte) ((p >> 8) & 0xFF));
                buf.put((byte) (p & 0xFF));
                buf.put((byte) ((p >> 24) & 0xFF));
            }
            buf.flip();

            id = glGenTextures();
            glBindTexture(GL_TEXTURE_2D, id);
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_S, wrapS);
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_WRAP_T, wrapT);
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MIN_FILTER, minFilter);
            glTexParameteri(GL_TEXTURE_2D, GL_TEXTURE_MAG_FILTER, magFilter);
            glPixelStorei(GL_UNPACK_ALIGNMENT, 1);
            glTexImage2D(GL_TEXTURE_2D, 0, GL_SRGB8_ALPHA8, width, height, 0,
                         GL_RGBA, GL_UNSIGNED_BYTE, buf);

            // Mipmaps нужны только если фильтр их использует
            boolean needsMipmaps = minFilter == GL_NEAREST_MIPMAP_NEAREST
                                || minFilter == GL_LINEAR_MIPMAP_NEAREST
                                || minFilter == GL_NEAREST_MIPMAP_LINEAR
                                || minFilter == GL_LINEAR_MIPMAP_LINEAR;
            if (needsMipmaps) glGenerateMipmap(GL_TEXTURE_2D);
            glBindTexture(GL_TEXTURE_2D, 0);
            System.out.println("Texture loaded: " + width + "x" + height
                + " mag=" + filterName(magFilter)
                + " min=" + filterName(minFilter));
        } catch (Exception e) {
            throw new RuntimeException("Texture load failed", e);
        }
    }

    public void bind(int unit) {
        glActiveTexture(GL_TEXTURE0 + unit);
        glBindTexture(GL_TEXTURE_2D, id);
    }

    public void cleanup() { glDeleteTextures(id); }

    private static String filterName(int f) {
        switch (f) {
            case GL_NEAREST: return "NEAREST";
            case GL_LINEAR: return "LINEAR";
            case GL_NEAREST_MIPMAP_NEAREST: return "NEAREST_MIPMAP_NEAREST";
            case GL_LINEAR_MIPMAP_NEAREST: return "LINEAR_MIPMAP_NEAREST";
            case GL_NEAREST_MIPMAP_LINEAR: return "NEAREST_MIPMAP_LINEAR";
            case GL_LINEAR_MIPMAP_LINEAR: return "LINEAR_MIPMAP_LINEAR";
            default: return "?(" + f + ")";
        }
    }
}
