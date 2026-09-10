package com.lodeen.engine.scene;

import com.lodeen.client.net.RemotePlayers;
import com.lodeen.engine.graphics.Mesh;
import com.lodeen.engine.graphics.ShaderProgram;
import com.lodeen.shared.util.PlayerColors;
import org.joml.Matrix4f;
import static org.lwjgl.opengl.GL30.*;

public class RemotePlayerRenderer {
    private final ShaderProgram shader;
    private final Mesh cube;

    public RemotePlayerRenderer() {
        shader = new ShaderProgram(VERT, FRAG);
        cube = new Mesh(CUBE_V, CUBE_I, new int[]{3, 3, 2});
    }

    public void render(Camera cam, int w, int h) {
        shader.use();
        setMat4("uView", cam.getView());
        setMat4("uProj", cam.getProjection(w, h));

        for (var p : RemotePlayers.sampleAll()) {
            float[] c = PlayerColors.rgbOf(p.id);
            glUniform3f(shader.getUniformLocation("uColor"), c[0], c[1], c[2]);

            Matrix4f base = new Matrix4f()
                .translate(p.x, p.y, p.z)
                .rotateY((float) Math.toRadians(p.yaw));

            drawPart(base, 0,  0,     0,     0.9f, 1.1f, 0.5f);
            drawPart(base, 0,  0.85f, 0,     0.55f, 0.55f, 0.55f);
            drawPart(base, 0,  0.85f, 0.35f, 0.14f, 0.14f, 0.28f);
        }
    }

    private void drawPart(Matrix4f base, float x, float y, float z,
                          float sx, float sy, float sz) {
        Matrix4f m = new Matrix4f(base)
            .translate(x, y, z)
            .scale(sx, sy, sz);
        setMat4("uModel", m);
        cube.draw();
    }

    public void cleanup() { cube.cleanup(); shader.cleanup(); }

    private void setMat4(String n, Matrix4f m) {
        try (var mem = org.lwjgl.system.MemoryStack.stackPush()) {
            glUniformMatrix4fv(shader.getUniformLocation(n), false, m.get(mem.mallocFloat(16)));
        }
    }

    private static final float[] CUBE_V = {
        // front z=+0.5
        -0.5f,-0.5f, 0.5f,  0,0,1,  0,0,
         0.5f,-0.5f, 0.5f,  0,0,1,  1,0,
         0.5f, 0.5f, 0.5f,  0,0,1,  1,1,
        -0.5f, 0.5f, 0.5f,  0,0,1,  0,1,
        // back z=-0.5
         0.5f,-0.5f,-0.5f,  0,0,-1, 0,0,
        -0.5f,-0.5f,-0.5f,  0,0,-1, 1,0,
        -0.5f, 0.5f,-0.5f,  0,0,-1, 1,1,
         0.5f, 0.5f,-0.5f,  0,0,-1, 0,1,
        // top y=+0.5
        -0.5f, 0.5f, 0.5f,  0,1,0,  0,0,
         0.5f, 0.5f, 0.5f,  0,1,0,  1,0,
         0.5f, 0.5f,-0.5f,  0,1,0,  1,1,
        -0.5f, 0.5f,-0.5f,  0,1,0,  0,1,
        // bottom y=-0.5
        -0.5f,-0.5f,-0.5f,  0,-1,0, 0,0,
         0.5f,-0.5f,-0.5f,  0,-1,0, 1,0,
         0.5f,-0.5f, 0.5f,  0,-1,0, 1,1,
        -0.5f,-0.5f, 0.5f,  0,-1,0, 0,1,
        // right x=+0.5
         0.5f,-0.5f, 0.5f,  1,0,0,  0,0,
         0.5f,-0.5f,-0.5f,  1,0,0,  1,0,
         0.5f, 0.5f,-0.5f,  1,0,0,  1,1,
         0.5f, 0.5f, 0.5f,  1,0,0,  0,1,
        // left x=-0.5
        -0.5f,-0.5f,-0.5f, -1,0,0,  0,0,
        -0.5f,-0.5f, 0.5f, -1,0,0,  1,0,
        -0.5f, 0.5f, 0.5f, -1,0,0,  1,1,
        -0.5f, 0.5f,-0.5f, -1,0,0,  0,1,
    };

    private static final int[] CUBE_I = {
        0,1,2, 0,2,3,      4,5,6, 4,6,7,
        8,9,10, 8,10,11,   12,13,14, 12,14,15,
        16,17,18, 16,18,19, 20,21,22, 20,22,23
    };

    private static final String VERT = """
        #version 330 core
        layout(location=0) in vec3 aPos;
        layout(location=1) in vec3 aNormal;
        layout(location=2) in vec2 aUV;
        uniform mat4 uModel;
        uniform mat4 uView;
        uniform mat4 uProj;
        out vec3 vNormal;
        void main() {
            vNormal = mat3(uModel) * aNormal;
            gl_Position = uProj * uView * uModel * vec4(aPos, 1.0);
        }
        """;

    private static final String FRAG = """
        #version 330 core
        in vec3 vNormal;
        out vec4 FragColor;
        uniform vec3 uColor;
        void main() {
            float l = 0.6 + 0.4 * max(dot(normalize(vNormal), vec3(0.4, 0.8, 0.4)), 0.0);
            FragColor = vec4(uColor * l, 1.0);
        }
        """;
}
