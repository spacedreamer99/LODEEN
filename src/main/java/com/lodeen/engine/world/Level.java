package com.lodeen.engine.world;

import java.util.ArrayList;
import java.util.List;
import static org.lwjgl.opengl.GL30.*;

public class Level {
    public static class Object3D {
        public String name;
        public float[] vertices;  // Данные в RAM (interleaved x,y,z)
        public int vao, vbo;
        public int vertexCount;
        public boolean uploaded;  // true — данные находятся в VRAM
    }

    private final List<Object3D> objects = new ArrayList<>();
    private boolean loaded = false;

    // Добавляет объект с вершинными данными (только RAM, без VRAM)
    public void addObject(String name, float[] vertices) {
        Object3D o = new Object3D();
        o.name = name;
        o.vertices = vertices;
        o.vertexCount = vertices.length / 3;
        o.uploaded = false;
        objects.add(o);
    }

    // Загружает уровень: отправляет все объекты из RAM в VRAM
    public void load() {
        if (loaded) return;
        for (Object3D o : objects) upload(o);
        loaded = true;
    }

    // Выгружает из VRAM, но оставляет данные в RAM для повторной загрузки
    public void unload() {
        for (Object3D o : objects) freeVRAM(o);
        loaded = false;
    }

    // Повторно загружает данные из RAM в VRAM
    public void reload() {
        load();
    }

    // Полная очистка: VRAM + RAM (вызывать при удалении уровня)
    public void dispose() {
        for (Object3D o : objects) freeVRAM(o);
        objects.clear();
        loaded = false;
    }

    // Загружает вершинные данные в GPU (VAO + VBO)
    private void upload(Object3D o) {
        if (o.uploaded) return;
        o.vao = glGenVertexArrays();
        o.vbo = glGenBuffers();
        glBindVertexArray(o.vao);
        glBindBuffer(GL_ARRAY_BUFFER, o.vbo);
        glBufferData(GL_ARRAY_BUFFER, o.vertices, GL_STATIC_DRAW);
        glVertexAttribPointer(0, 3, GL_FLOAT, false, 0, 0);
        glEnableVertexAttribArray(0);
        glBindVertexArray(0);
        o.uploaded = true;
    }

    // Освобождает VRAM для объекта, но сохраняет данные в RAM
    private void freeVRAM(Object3D o) {
        if (!o.uploaded) return;
        glDeleteBuffers(o.vbo);
        glDeleteVertexArrays(o.vao);
        o.vao = 0;
        o.vbo = 0;
        o.uploaded = false;
    }

    public boolean isLoaded() { return loaded; }
    public int getObjectCount() { return objects.size(); }
    public List<Object3D> getObjects() { return objects; }
}
