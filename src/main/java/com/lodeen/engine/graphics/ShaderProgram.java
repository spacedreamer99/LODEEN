package com.lodeen.engine.graphics;

import static org.lwjgl.opengl.GL30.*;

public class ShaderProgram {
    private int id;
    public ShaderProgram(String vertexSrc, String fragmentSrc) {
        int vs = compileShader(GL_VERTEX_SHADER, vertexSrc);
        int fs = compileShader(GL_FRAGMENT_SHADER, fragmentSrc);
        id = glCreateProgram();
        glAttachShader(id, vs);
        glAttachShader(id, fs);
        glLinkProgram(id);
        if (glGetProgrami(id, GL_LINK_STATUS) == GL_FALSE) {
            throw new RuntimeException("Program link error: " + glGetProgramInfoLog(id));
        }
        glDeleteShader(vs);
        glDeleteShader(fs);
    }
    public void use() { glUseProgram(id); }
    public int getId() { return id; }
    public void setInt(String name, int value) {
        glUniform1i(glGetUniformLocation(id, name), value);
    }
    public void setFloat(String name, float value) {
        glUniform1f(glGetUniformLocation(id, name), value);
    }
    public void cleanup() { glDeleteProgram(id); }

    private int compileShader(int type, String src) {
        int shader = glCreateShader(type);
        glShaderSource(shader, src);
        glCompileShader(shader);
        if (glGetShaderi(shader, GL_COMPILE_STATUS) == GL_FALSE) {
            throw new RuntimeException("Shader compile error: " + glGetShaderInfoLog(shader));
        }
        return shader;
    }
}