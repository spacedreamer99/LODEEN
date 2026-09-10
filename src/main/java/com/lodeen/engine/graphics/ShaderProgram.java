package com.lodeen.engine.graphics;

import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import static org.lwjgl.opengl.GL30.*;

public class ShaderProgram {
    private int id;

    public ShaderProgram(String v, String f) {
        int vs = compile(GL_VERTEX_SHADER, resolve(v));
        int fs = compile(GL_FRAGMENT_SHADER, resolve(f));
        id = glCreateProgram();
        glAttachShader(id, vs); glAttachShader(id, fs); glLinkProgram(id);
        if (glGetProgrami(id, GL_LINK_STATUS) == GL_FALSE)
            throw new RuntimeException("Program link: " + glGetProgramInfoLog(id));
        glDeleteShader(vs); glDeleteShader(fs);
    }

    public static ShaderProgram fromResources(String v, String f) {
        return new ShaderProgram(read(v), read(f));
    }

    // Раскрывает `// #include "file.glsl"` в содержимое файла из ресурсов
    private static String resolve(String src) {
        StringBuilder out = new StringBuilder();
        for (String line : src.split("\n")) {
            String t = line.trim();
            if (t.startsWith("// #include")) {
                String name = t.substring(t.indexOf('"') + 1, t.lastIndexOf('"'));
                out.append(read("/shaders/" + name));
            } else {
                out.append(line);
            }
            out.append('\n');
        }
        return out.toString();
    }

    private static String read(String path) {
        try (InputStream is = ShaderProgram.class.getResourceAsStream(path)) {
            if (is == null) throw new RuntimeException("Shader missing: " + path);
            return new String(is.readAllBytes(), StandardCharsets.UTF_8);
        } catch (Exception e) { throw new RuntimeException(e); }
    }

    public void use() { glUseProgram(id); }
    public int getId() { return id; }
    public void setInt(String n, int v) { glUniform1i(glGetUniformLocation(id, n), v); }
    public void setFloat(String n, float v) { glUniform1f(glGetUniformLocation(id, n), v); }
    public void cleanup() { glDeleteProgram(id); }

    private int compile(int type, String src) {
        int s = glCreateShader(type);
        glShaderSource(s, src); glCompileShader(s);
        if (glGetShaderi(s, GL_COMPILE_STATUS) == GL_FALSE)
            throw new RuntimeException("Shader: " + glGetShaderInfoLog(s));
        return s;
    }
}
