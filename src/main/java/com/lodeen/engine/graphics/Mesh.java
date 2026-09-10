package com.lodeen.engine.graphics;

import static org.lwjgl.opengl.GL30.*;

public class Mesh {
    private int vao, vbo, ebo, indexCount;
    private float boundingRadius = 1f;
    public Material material = new Material();

    public Mesh(float[] vertices, int[] indices, int[] attribSizes) {
        indexCount = indices.length;
        vao = glGenVertexArrays();
        vbo = glGenBuffers();
        ebo = glGenBuffers();

        glBindVertexArray(vao);
        glBindBuffer(GL_ARRAY_BUFFER, vbo);
        glBufferData(GL_ARRAY_BUFFER, vertices, GL_STATIC_DRAW);
        glBindBuffer(GL_ELEMENT_ARRAY_BUFFER, ebo);
        glBufferData(GL_ELEMENT_ARRAY_BUFFER, indices, GL_STATIC_DRAW);

        int strideFloats = 0;
        for (int s : attribSizes) strideFloats += s;
        int strideBytes = strideFloats * Float.BYTES;

        long offset = 0;
        for (int i = 0; i < attribSizes.length; i++) {
            glVertexAttribPointer(i, attribSizes[i], GL_FLOAT, false, strideBytes, offset);
            glEnableVertexAttribArray(i);
            offset += (long) attribSizes[i] * Float.BYTES;
        }
        glBindVertexArray(0);

        float maxSq = 0f;
        for (int i = 0; i < vertices.length; i += strideFloats) {
            float x = vertices[i], y = vertices[i+1], z = vertices[i+2];
            float sq = x*x + y*y + z*z;
            if (sq > maxSq) maxSq = sq;
        }
        boundingRadius = (float) Math.sqrt(maxSq);
    }

    public void draw() {
        glBindVertexArray(vao);
        glDrawElements(GL_TRIANGLES, indexCount, GL_UNSIGNED_INT, 0);
        glBindVertexArray(0);
    }

    public float getBoundingRadius() { return boundingRadius; }

    public void cleanup() {
        glDeleteBuffers(vbo);
        glDeleteBuffers(ebo);
        glDeleteVertexArrays(vao);
    }
}
