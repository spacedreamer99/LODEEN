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
            "    vec2 uv = gl_FragCoord.xy / resolution.y;\n" +
            "    float density = 8000.0;\n" +
            "    vec2 cell = floor(uv * density);\n" +
            "    vec2 local = fract(uv * density) - 0.5;\n" +
            "    vec4 rand = vec4(\n" +
            "        hash(cell),\n" +
            "        hash(cell + 1.0),\n" +
            "        hash(cell + 2.0),\n" +
            "        hash(cell + 3.0)\n" +
            "    );\n" +
            "    vec2 starPos = (rand.xy - 0.5) * 0.9;\n" +
            "    float d = length(local - starPos);\n" +
            "    float classRand = rand.z;\n" +  // определяет спектральный класс
            "    vec3 color;\n" +
            "    float sizeMult;\n" +
            "    // Вероятности: M=52%, K=20%, G=12%, F=8%, A=5%, B=2%, O=1%\n" +
            "    if (classRand > 0.48) {\n" +           // M (красный карлик)
            "        color = vec3(1.0, 0.6, 0.3);\n" +
            "        sizeMult = 1.0;\n" +
            "    } else if (classRand > 0.28) {\n" +   // K (оранжевый)
            "        color = vec3(1.0, 0.8, 0.6);\n" +
            "        sizeMult = 1.3;\n" +
            "    } else if (classRand > 0.16) {\n" +   // G (жёлтый)
            "        color = vec3(1.0, 0.95, 0.8);\n" +
            "        sizeMult = 1.6;\n" +
            "    } else if (classRand > 0.08) {\n" +   // F (бело-жёлтый)
            "        color = vec3(0.9, 0.9, 1.0);\n" +
            "        sizeMult = 2.0;\n" +
            "    } else if (classRand > 0.03) {\n" +   // A (белый)
            "        color = vec3(0.7, 0.8, 1.0);\n" +
            "        sizeMult = 2.8;\n" +
            "    } else if (classRand > 0.01) {\n" +   // B (голубой)
            "        color = vec3(0.5, 0.6, 1.0);\n" +
            "        sizeMult = 4.0;\n" +
            "    } else {\n" +                         // O (ярко-голубой)
            "        color = vec3(0.3, 0.4, 1.0);\n" +
            "        sizeMult = 6.0;\n" +
            "    }\n" +
            "    // Базовый минимальный размер 0.012 (~8px), множитель класса увеличивает его\n" +
            "    float size = 0.012 * sizeMult;\n" +
            "    // Яркость зависит от размера (имитация светимости)\n" +
            "    float brightness = smoothstep(size, 0.0, d) * (0.5 + size * 8.0);\n" +
            "    // Добавляем яркое ядро для крупных звёзд\n" +
            "    brightness += smoothstep(size * 0.3, 0.0, d) * step(0.92, rand.w) * 1.5;\n" +
            "    FragColor = vec4(color * brightness, 1.0);\n" +
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
