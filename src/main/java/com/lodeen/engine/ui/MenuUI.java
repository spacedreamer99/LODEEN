package com.lodeen.engine.ui;

import com.lodeen.engine.graphics.RectRenderer;
import com.lodeen.engine.graphics.StarFieldRenderer;
import com.lodeen.engine.graphics.TextRenderer;
import static org.lwjgl.glfw.GLFW.*;

public class MenuUI {
    private StarFieldRenderer starField;
    private RectRenderer rectRenderer;
    private TextRenderer textRenderer;
    private long window;
    private boolean fullscreen = false;
    private int windowedX, windowedY, windowedW, windowedH;
    private String[] labels = {"Singleplayer", "Multiplayer", "Fullscreen", "Encyclopedia", "Settings", "Exit"};
    private int hovered = -1;
    private static final double FULLSCREEN_COOLDOWN = 0.2;
    private double lastFullscreenToggleTime = 0.0;

    public MenuUI(long window) {
        this.window = window;
        starField = new StarFieldRenderer();
        rectRenderer = new RectRenderer();
        textRenderer = new TextRenderer();
    }

    public void render(float time) {
        int[] w = new int[1], h = new int[1];
        glfwGetWindowSize(window, w, h);
        int width = w[0], height = h[0];

        // Рисуем звёздный фон
        starField.render(width, height);

        // Параметры кнопок
        float buttonW = 200 * (width / 1280.0f);
        float buttonH = 40 * (height / 720.0f);
        float spacing = 10 * (height / 720.0f);
        float totalH = labels.length * (buttonH + spacing) - spacing;
        float startX = (width - buttonW) / 2;
        float startY = (height - totalH) / 2;

        // Координаты мыши (Y не инвертируем, т.к. кнопки считаются от верхнего края)
        double[] mx = new double[1], my = new double[1];
        glfwGetCursorPos(window, mx, my);
        float mouseX = (float) mx[0];
        float mouseY = (float) my[0];

        hovered = -1;
        for (int i = 0; i < labels.length; i++) {
            float y = startY + i * (buttonH + spacing);
            boolean inside = mouseX >= startX && mouseX <= startX + buttonW &&
                             mouseY >= y && mouseY <= y + buttonH;
            if (inside) hovered = i;

            // Отрисовка прямоугольника кнопки
            float r = 0.2f, g = 0.2f, b = 0.2f, a = 0.6f;
            if (inside) {
                r = 0.4f; g = 0.4f; b = 0.4f; a = 0.8f;
            }
            rectRenderer.draw(startX, y, buttonW, buttonH, width, height, r, g, b, a);

            // Отрисовка текста
            float textScale = buttonH * 0.02f;
            float centerX = startX + buttonW / 2;
            float centerY = y + buttonH / 2;
            textRenderer.drawTextCentered(labels[i], centerX, centerY, textScale, width, height);
        }

        // Обработка клика
        if (glfwGetMouseButton(window, GLFW_MOUSE_BUTTON_LEFT) == GLFW_PRESS && hovered >= 0) {
            handleClick(hovered);
        }
    }

    private void handleClick(int index) {
        switch (index) {
            case 0: System.out.println("Singleplayer placeholder"); break;
            case 1: System.out.println("Multiplayer placeholder"); break;
            case 2:
                double now = glfwGetTime();
                if (now - lastFullscreenToggleTime >= FULLSCREEN_COOLDOWN) {
                    lastFullscreenToggleTime = now;
                    toggleFullscreen();
                }
                break;
            case 3: System.out.println("Encyclopedia placeholder"); break;
            case 4: System.out.println("Settings placeholder"); break;
            case 5: glfwSetWindowShouldClose(window, true); break;
        }
    }

    private void toggleFullscreen() {
        fullscreen = !fullscreen;
        int[] w = new int[1], h = new int[1];
        if (fullscreen) {
            int[] x = new int[1], y = new int[1];
            glfwGetWindowPos(window, x, y);
            glfwGetWindowSize(window, w, h);
            windowedX = x[0]; windowedY = y[0]; windowedW = w[0]; windowedH = h[0];
            long monitor = glfwGetPrimaryMonitor();
            var vm = glfwGetVideoMode(monitor);
            glfwSetWindowMonitor(window, monitor, 0, 0, vm.width(), vm.height(), GLFW_DONT_CARE);
        } else {
            glfwSetWindowMonitor(window, 0, windowedX, windowedY, windowedW, windowedH, GLFW_DONT_CARE);
        }
    }

    public void cleanup() {
        starField.cleanup();
        rectRenderer.cleanup();
        textRenderer.cleanup();
    }
}
