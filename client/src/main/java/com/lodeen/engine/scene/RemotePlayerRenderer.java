package com.lodeen.engine.scene;

import com.lodeen.client.net.RemotePlayers;
import com.lodeen.engine.graphics.Mesh;
import com.lodeen.engine.graphics.ShaderProgram;
import org.joml.Matrix4f;

import static org.lwjgl.opengl.GL30.*;

public class RemotePlayerRenderer {
    private final ShaderProgram shader;
    private final Mesh mesh;

    public RemotePlayerRenderer() {
        shader = new ShaderProgram(VERT, FRAG);
        mesh = buildOctahedron();
    }

    public void render(Camera cam, int w, int h) {
        shader.use();
        setMat4("uView", cam.getView());
        setMat4("uProj", cam.getProjection(w, h));
        for (var p : RemotePlayers.all()) {
            Matrix4f model = new Matrix4f().translate(p.x, p.y, p.z).scale(0.3f);
            setMat4("uModel", model);
            mesh.draw();
        }
    }

    public void cleanup() { mesh.cleanup(); shader.cleanup(); }

    private void setMat4(String n, Matrix4f m) {
        try (var mem = org.lwjgl.system.MemoryStack.stackPush()) {
            glUniformMatrix4fv(shader.getUniformLocation(n), false, m.get(mem.mallocFloat(16)));
        }
    }

    private static Mesh buildOctahedron() {
        // 6 вершин, 8 треугольников
        float[] v = {
            0, 1, 0,   0, 1, 0,   0, 0,   // top
            0,-1, 0,   0,-1, 0,   0, 0,   // bottom
            1, 0, 0,   1, 0, 0,   0, 0,   // right
           -1, 0, 0,  -1, 0, 0,   0, 0,   // left
            0, 0, 1,   0, 0, 1,   0, 0,   // front
            0, 0,-1,   0, 0,-1,   0, 0    // back
        };
        int[] idx = {
            0,2,4, 0,4,3, 0,3,5, 0,5,2,
            1,4,2, 1,3,4, 1,5,3, 1,2,5
        };
        return new Mesh(v, idx, new int[]{3, 3, 2});
    }

    private static final String VERT = """
        #version 330 core
        layout(location=0) in vec3 aPos;
        layout(location=1) in vec3 aNormal;
        layout(location=2) in vec2 aUV;
        uniform mat4 uModel;
        uniform mat4 uView;
        uniform mat4 uProj;
        void main() {
            gl_Position = uProj * uView * uModel * vec4(aPos, 1.0);
        }
        """;

    private static final String FRAG = """
        #version 330 core
        out vec4 FragColor;
        void main() {
            FragColor = vec4(0.2, 0.8, 1.0, 1.0);
        }
        """;
}
