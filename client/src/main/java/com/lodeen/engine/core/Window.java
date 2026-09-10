package com.lodeen.engine.core;

import com.lodeen.client.net.NetworkManager;
import com.lodeen.engine.scene.Scene;
import com.lodeen.engine.ui.DebugOverlay;
import com.lodeen.engine.ui.ImGuiLayer;
import com.lodeen.engine.ui.MenuUI;
import com.lodeen.game.SettingsManager;
import com.lodeen.game.SettingsUI;
import org.lwjgl.glfw.GLFWErrorCallback;
import static org.lwjgl.glfw.GLFW.*;
import static org.lwjgl.opengl.GL11.*;

public class Window {
    private enum GameState { MENU, PLAYING }

    private final long handle;
    private int width, height;
    private MenuUI menuUI;
    private Scene scene;
    private GameState state = GameState.MENU;
    private ImGuiLayer imgui;
    private DebugOverlay overlay;
    private NetworkManager net;
    private float lastDt = 0.016f;

    public Window(String title, int w, int h) {
        WindowInit init = new WindowInit(title, w, h);
        this.handle = init.handle;
        this.width = init.width;
        this.height = init.height;

        menuUI = new MenuUI(handle);
        imgui = new ImGuiLayer();
        imgui.init(handle);
        overlay = new DebugOverlay();

        net = new NetworkManager();
        net.connect("localhost", 25565, "Player");
    }

    public void loop() {
        double last = glfwGetTime();
        while (!glfwWindowShouldClose(handle)) {
            double now = glfwGetTime();
            float dt = (float) (now - last);
            last = now; lastDt = dt;

            int[] w = new int[1], h = new int[1];
            glfwGetWindowSize(handle, w, h);
            width = w[0]; height = h[0];

            overlay.toggle(handle);
            net.tick();

            switch (state) {
                case MENU -> tickMenu(now);
                case PLAYING -> tickPlaying(dt);
            }

            imgui.startFrame();
            if (state == GameState.MENU) menuUI.renderWidgets();
            overlay.render(1.0f / Math.max(0.0001f, lastDt), state.name(), width, height, net);
            if (SettingsManager.isSettingsVisible()) SettingsUI.render(width, height);
            imgui.endFrame();

            glfwSwapBuffers(handle);
            glfwPollEvents();
        }
        shutdown();
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
            scene.init(handle, net);
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

    private void shutdown() {
        if (scene != null) scene.cleanup();
        if (net != null) net.disconnect();
        if (menuUI != null) menuUI.cleanup();
        if (imgui != null) imgui.dispose();
        glfwDestroyWindow(handle);
        glfwTerminate();
        GLFWErrorCallback cb = glfwSetErrorCallback(null);
        if (cb != null) cb.free();
    }
}
