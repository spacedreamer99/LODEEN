package com.lodeen.engine.ui;

import com.lodeen.engine.graphics.RectRenderer;
import com.lodeen.engine.graphics.StarFieldRenderer;
import com.lodeen.engine.graphics.TextRenderer;
import static org.lwjgl.glfw.GLFW.*;

public class MenuUI {
    private final StarFieldRenderer starField = new StarFieldRenderer();
    private final RectRenderer rect = new RectRenderer();
    private final TextRenderer text = new TextRenderer();
    private final long window;
    private boolean fullscreen = false;
    private int wx, wy, ww, wh;
    private final String[] labels = {"Singleplayer","Multiplayer","Fullscreen","Encyclopedia","Settings","Exit"};
    private int hovered = -1;
    private boolean startRequested = false;
    private double lastToggle = 0;
    private boolean wasMouseDown = false;

    public MenuUI(long window) { this.window = window; }

    public void render(float time) {
        int[] w = new int[1], h = new int[1];
        glfwGetWindowSize(window, w, h);
        int W = w[0], H = h[0];
        starField.render(W, H);

        float bw = 600 * (W / 1280f), bh = 90 * (H / 720f), sp = 14 * (H / 720f);
        float totalH = labels.length * (bh + sp) - sp;
        float sx = (W - bw) / 2, sy = (H - totalH) / 2;

        double[] mx = new double[1], my = new double[1];
        glfwGetCursorPos(window, mx, my);
        float mX = (float) mx[0], mY = (float) my[0];

        hovered = -1;
        for (int i = 0; i < labels.length; i++) {
            float y = sy + i * (bh + sp);
            boolean inside = mX >= sx && mX <= sx + bw && mY >= y && mY <= y + bh;
            if (inside) hovered = i;
            float r = inside ? 0.4f : 0.2f, g = r, b = r, a = inside ? 0.8f : 0.6f;
            rect.draw(sx, y, bw, bh, W, H, r, g, b, a);
            text.drawTextCentered(labels[i], sx + bw / 2, y + bh / 2,
                                  (int) (bh * 0.55f), W, H);
        }

        // edge-detection: реагируем только на переход up -> down
        boolean isDown = glfwGetMouseButton(window, GLFW_MOUSE_BUTTON_LEFT) == GLFW_PRESS;
        if (isDown && !wasMouseDown && hovered >= 0) handleClick(hovered);
        wasMouseDown = isDown;
    }

    private void handleClick(int i) {
        switch (i) {
            case 0: startRequested = true; break;
            case 1: System.out.println("Multiplayer (TODO)"); break;
            case 2:
                double now = glfwGetTime();
                if (now - lastToggle > 0.2) { lastToggle = now; toggleFullscreen(); }
                break;
            case 3: System.out.println("Encyclopedia (TODO)"); break;
            case 4: System.out.println("Settings (TODO)"); break;
            case 5: glfwSetWindowShouldClose(window, true); break;
        }
    }

    public boolean consumeStartRequest() {
        if (startRequested) { startRequested = false; return true; }
        return false;
    }

    private void toggleFullscreen() {
        fullscreen = !fullscreen;
        int[] w = new int[1], h = new int[1];
        if (fullscreen) {
            int[] x = new int[1], y = new int[1];
            glfwGetWindowPos(window, x, y);
            glfwGetWindowSize(window, w, h);
            wx = x[0]; wy = y[0]; ww = w[0]; wh = h[0];
            long m = glfwGetPrimaryMonitor();
            var vm = glfwGetVideoMode(m);
            glfwSetWindowMonitor(window, m, 0, 0, vm.width(), vm.height(), GLFW_DONT_CARE);
        } else {
            glfwSetWindowMonitor(window, 0, wx, wy, ww, wh, GLFW_DONT_CARE);
        }
    }

    public void cleanup() {
        starField.cleanup(); rect.cleanup(); text.cleanup();
    }
}
