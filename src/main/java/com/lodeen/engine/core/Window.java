package com.lodeen.engine.core;

import com.lodeen.engine.scene.Scene;
import com.lodeen.engine.ui.ImGuiLayer;
import com.lodeen.engine.ui.MenuUI;
import imgui.ImGui;
import imgui.flag.ImGuiCond;
import org.lwjgl.glfw.GLFWErrorCallback;
import org.lwjgl.glfw.GLFWVidMode;
import org.lwjgl.opengl.GL;
import org.lwjgl.system.MemoryUtil;

import static org.lwjgl.glfw.GLFW.*;
import static org.lwjgl.opengl.GL11.*;

public class Window {
    private enum GameState { MENU, PLAYING }

    private long handle;
    private int width, height;
    private final String title;
    private MenuUI menuUI;
    private Scene scene;
    private GameState state = GameState.MENU;
    private ImGuiLayer imgui;
    private float lastDt = 0.016f;
    private boolean debugVisible = false;
    private boolean f3WasPressed = false;

    public Window(String title, int width, int height) {
        this.title = title; this.width = width; this.height = height;
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

        long monitor = glfwGetPrimaryMonitor();
        GLFWVidMode mode = monitor != MemoryUtil.NULL ? glfwGetVideoMode(monitor) : null;
        if (mode != null) { width = mode.width(); height = mode.height(); }

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
            width = w; height = h;
            glViewport(0, 0, w, h);
        });

        menuUI = new MenuUI(handle);
        imgui = new ImGuiLayer();
        imgui.init(handle);
    }

    public void loop() {
        double last = glfwGetTime();
        while (!glfwWindowShouldClose(handle)) {
            double now = glfwGetTime();
            float dt = (float) (now - last);
            last = now;
            lastDt = dt;

            int[] w = new int[1], h = new int[1];
            glfwGetWindowSize(handle, w, h);
            width = w[0]; height = h[0];

            updateF3();

            switch (state) {
                case MENU -> tickMenu(now);
                case PLAYING -> tickPlaying(dt);
            }

            imgui.startFrame();
            if (state == GameState.MENU) {
                menuUI.renderWidgets();
            }
            if (debugVisible) renderDebugOverlay();
            imgui.endFrame();

            glfwSwapBuffers(handle);
            glfwPollEvents();
        }
        if (scene != null) scene.cleanup();
        if (menuUI != null) menuUI.cleanup();
        if (imgui != null) imgui.dispose();
        glfwDestroyWindow(handle);
        glfwTerminate();
        GLFWErrorCallback cb = glfwSetErrorCallback(null);
        if (cb != null) cb.free();
    }

    private void updateF3() {
        boolean pressed = glfwGetKey(handle, GLFW_KEY_F3) == GLFW_PRESS;
        if (pressed && !f3WasPressed) debugVisible = !debugVisible;
        f3WasPressed = pressed;
    }

    private void renderDebugOverlay() {
        ImGui.setNextWindowPos(10, 10, ImGuiCond.Always);
        ImGui.setNextWindowSize(280, 140, ImGuiCond.Always);
        ImGui.begin("Debug (F3)");
        ImGui.text("FPS: " + (int) (1.0f / Math.max(0.0001f, lastDt)));
        ImGui.text("State: " + state);
        ImGui.text("Resolution: " + width + " x " + height);
        ImGui.end();
    }

    private void tickMenu(double now) {
        glEnable(GL_BLEND);
        glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
        glViewport(0, 0, width, height);
        glClearColor(0, 0, 0, 1);
        glClear(GL_COLOR_BUFFER_BIT);
        menuUI.renderBackground(width, height);
        if (menuUI.consumeStartRequest()) {
            scene = new Scene();
            scene.init(handle);
            state = GameState.PLAYING;
        }
    }

    private void tickPlaying(float dt) {
        scene.update(dt);
        scene.render(width, height);
        if (scene.shouldExit()) {
            scene.cleanup();
            scene = null;
            state = GameState.MENU;
        }
    }
}
