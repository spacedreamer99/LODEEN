package com.lodeen.engine.core;

import org.lwjgl.glfw.GLFWErrorCallback;
import org.lwjgl.glfw.GLFWVidMode;
import org.lwjgl.opengl.GL;
import org.lwjgl.system.MemoryUtil;
import static org.lwjgl.glfw.GLFW.*;
import static org.lwjgl.opengl.GL11.*;

public class WindowInit {
    public long handle;
    public int width, height;

    public WindowInit(String title, int defW, int defH) {
        GLFWErrorCallback.createPrint(System.err).set();
        if (!glfwInit()) throw new IllegalStateException("Cannot init GLFW");
        glfwDefaultWindowHints();
        glfwWindowHint(GLFW_VISIBLE, GLFW_FALSE);
        glfwWindowHint(GLFW_RESIZABLE, GLFW_TRUE);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MAJOR, 3);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MINOR, 3);
        glfwWindowHint(GLFW_OPENGL_PROFILE, GLFW_OPENGL_CORE_PROFILE);

        long monitor = glfwGetPrimaryMonitor();
        GLFWVidMode mode = monitor != MemoryUtil.NULL ? glfwGetVideoMode(monitor) : null;
        width  = mode != null ? mode.width()  : defW;
        height = mode != null ? mode.height() : defH;

        handle = glfwCreateWindow(width, height, title, monitor, MemoryUtil.NULL);
        if (handle == MemoryUtil.NULL) throw new RuntimeException("Window creation failed");
        glfwMakeContextCurrent(handle);
        glfwSwapInterval(1);
        glfwShowWindow(handle);
        GL.createCapabilities();

        glViewport(0, 0, width, height);
        glEnable(GL_BLEND);
        glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
        glClearColor(0, 0, 0, 1);

        glfwSetFramebufferSizeCallback(handle, (win, w, h) -> {
            this.width = w; this.height = h;
            glViewport(0, 0, w, h);
        });
    }
}
