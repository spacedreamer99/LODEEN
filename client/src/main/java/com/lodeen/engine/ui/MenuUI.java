package com.lodeen.engine.ui;

import com.lodeen.engine.graphics.StarFieldRenderer;
import com.lodeen.game.SettingsManager;
import com.lodeen.game.SettingsUI;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import imgui.ImGui;
import imgui.flag.ImGuiCond;
import imgui.flag.ImGuiWindowFlags;
import org.lwjgl.system.MemoryUtil;
import static org.lwjgl.glfw.GLFW.*;

public class MenuUI {
    private static final Logger log = LoggerFactory.getLogger(MenuUI.class);

    private final StarFieldRenderer starField = new StarFieldRenderer();
    private final long window;
    private boolean startRequested = false;
    private int wx = 100, wy = 100, ww = 1280, wh = 720;
    private double lastToggle = 0;

    public MenuUI(long window) { this.window = window; }

    /** Рендер звёздного фона через OpenGL — вызывается ДО imgui.startFrame(). */
    public void renderBackground(int w, int h) {
        starField.render(w, h);
    }

    /** Рендер кнопок через ImGui — вызывается МЕЖДУ imgui.startFrame() и endFrame(). */
    public void renderWidgets() {
        int[] w = new int[1], h = new int[1];
        glfwGetWindowSize(window, w, h);
        int W = w[0], H = h[0];

        float panelW = 500, panelH = 520;
        ImGui.setNextWindowPos(W * 0.5f, H * 0.5f, ImGuiCond.Always, 0.5f, 0.5f);
        ImGui.setNextWindowSize(panelW, panelH, ImGuiCond.Always);

        int flags = ImGuiWindowFlags.NoTitleBar
                  | ImGuiWindowFlags.NoResize
                  | ImGuiWindowFlags.NoMove
                  | ImGuiWindowFlags.NoCollapse
                  | ImGuiWindowFlags.NoBackground;

        ImGui.begin("MainMenu", flags);

        float btnW = 360, btnH = 55;
        float offsetX = (panelW - btnW) / 2;

        // Заголовок
        ImGui.setCursorPos(offsetX, 20);
        ImGui.text("LODEEN");

        ImGui.setCursorPos(offsetX, 110);

        if (ImGui.button("Singleplayer", btnW, btnH)) startRequested = true;
        ImGui.setCursorPosX(offsetX);
        if (ImGui.button("Multiplayer", btnW, btnH)) log.info("Multiplayer clicked (TODO)");
        ImGui.setCursorPosX(offsetX);
        if (ImGui.button("Fullscreen", btnW, btnH)) {
            double now = glfwGetTime();
            if (now - lastToggle > 0.2) { lastToggle = now; toggleFullscreen(); }
        }
        ImGui.setCursorPosX(offsetX);
        if (ImGui.button("Encyclopedia", btnW, btnH)) log.info("Encyclopedia clicked (TODO)");
        ImGui.setCursorPosX(offsetX);
        if (ImGui.button("Settings", btnW, btnH)) SettingsManager.openSettings();
        ImGui.setCursorPosX(offsetX);
        if (ImGui.button("Exit", btnW, btnH)) glfwSetWindowShouldClose(window, true);

        ImGui.end();
    }

    private boolean isFullscreen() {
        return glfwGetWindowMonitor(window) != MemoryUtil.NULL;
    }

    private void toggleFullscreen() {
        if (isFullscreen()) {
            glfwSetWindowMonitor(window, MemoryUtil.NULL, wx, wy, ww, wh, GLFW_DONT_CARE);
        } else {
            int[] x = new int[1], y = new int[1], w = new int[1], h = new int[1];
            glfwGetWindowPos(window, x, y);
            glfwGetWindowSize(window, w, h);
            wx = x[0]; wy = y[0]; ww = w[0]; wh = h[0];
            long m = glfwGetPrimaryMonitor();
            if (m == MemoryUtil.NULL) return;
            var vm = glfwGetVideoMode(m);
            if (vm == null) return;
            glfwSetWindowMonitor(window, m, 0, 0, vm.width(), vm.height(), GLFW_DONT_CARE);
        }
    }

    public boolean consumeStartRequest() {
        if (startRequested) { startRequested = false; return true; }
        return false;
    }

    public void cleanup() {
        starField.cleanup();
    }
}
