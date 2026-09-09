package com.lodeen.engine.core;

import com.lodeen.engine.ui.MenuUI;
import org.lwjgl.glfw.GLFW;
import org.lwjgl.glfw.GLFWErrorCallback;
import org.lwjgl.opengl.GL;
import org.lwjgl.system.MemoryUtil;

import static org.lwjgl.glfw.GLFW.*;
import static org.lwjgl.opengl.GL11.*;

public class Window {
    private long handle;
    private int width, height;
    private String title;
    private MenuUI menuUI;

    public Window(String title, int width, int height) {
        this.title = title;
        this.width = width;
        this.height = height;
    }

    public void init() {
        GLFWErrorCallback.createPrint(System.err).set();
        if (!glfwInit()) throw new IllegalStateException("Cannot init GLFW");
        glfwDefaultWindowHints();
        glfwWindowHint(GLFW_VISIBLE, GLFW_FALSE);
        glfwWindowHint(GLFW_RESIZABLE, GLFW_TRUE);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MAJOR, 3);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MINOR, 3);
        glfwWindowHint(GLFW_OPENGL_PROFILE, GLFW_OPENGL_CORE_PROFILE);
        handle = glfwCreateWindow(width, height, title, MemoryUtil.NULL, MemoryUtil.NULL);
        if (handle == MemoryUtil.NULL) throw new RuntimeException("Failed to create window");
        glfwMakeContextCurrent(handle);
        glfwSwapInterval(1);
        glfwShowWindow(handle);
        GL.createCapabilities();

        // Включаем альфа-блендинг для прозрачных элементов UI
        glEnable(GL_BLEND);
        glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

        glClearColor(0, 0, 0, 1);

        // Устанавливаем колбэк изменения размера окна
        glfwSetFramebufferSizeCallback(handle, (win, newWidth, newHeight) -> {
            width = newWidth;
            height = newHeight;
            glViewport(0, 0, width, height);
        });

        menuUI = new MenuUI(handle);
    }

    public void loop() {
        double lastTime = glfwGetTime();
        while (!glfwWindowShouldClose(handle)) {
            double now = glfwGetTime();
            float time = (float) now;

            // Получаем актуальные размеры окна
            int[] w = new int[1], h = new int[1];
            glfwGetWindowSize(handle, w, h);
            width = w[0]; height = h[0];

            glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
            menuUI.render(time);

            glfwSwapBuffers(handle);
            glfwPollEvents();
        }
        cleanup();
    }

    private void cleanup() {
        if (menuUI != null) menuUI.cleanup();
        glfwDestroyWindow(handle);
        glfwTerminate();
        GLFWErrorCallback callback = glfwSetErrorCallback(null);
        if (callback != null) callback.free();
    }
}
