package com.lodeen.engine.scene;

import static org.lwjgl.glfw.GLFW.*;

public class InputController {
    private final long window;
    private final Camera camera;
    private double lastMX, lastMY;
    private boolean firstMouse = true;
    private float sensitivity = 0.12f;
    private boolean exitRequested = false;

    public InputController(long window, Camera camera) {
        this.window = window;
        this.camera = camera;
        captureMouse(true);
        glfwSetScrollCallback(window, (win, xoff, yoff) ->
            camera.multiplyThrottle((float) java.lang.Math.pow(1.15, yoff)));
    }

    public void update(float dt) {
        applyMouse();

        float fwd = axis(GLFW_KEY_W, GLFW_KEY_S);
        float rgt = axis(GLFW_KEY_D, GLFW_KEY_A);
        float up  = axis(GLFW_KEY_SPACE, GLFW_KEY_LEFT_SHIFT);
        float p   = axis(GLFW_KEY_UP, GLFW_KEY_DOWN);
        float y   = axis(GLFW_KEY_RIGHT, GLFW_KEY_LEFT);
        float r   = axis(GLFW_KEY_E, GLFW_KEY_Q);   // E = +roll, Q = -roll

        camera.update(dt, fwd, rgt, up, p, y, r);

        if (key(GLFW_KEY_ESCAPE)) exitRequested = true;
        if (key(GLFW_KEY_R)) camera.resetRotation();
    }

    private void applyMouse() {
        double[] mx = new double[1], my = new double[1];
        glfwGetCursorPos(window, mx, my);
        if (firstMouse) { lastMX = mx[0]; lastMY = my[0]; firstMouse = false; }
        float dx = (float) ((mx[0] - lastMX) * sensitivity);
        float dy = (float) ((my[0] - lastMY) * sensitivity);
        lastMX = mx[0]; lastMY = my[0];
        if (dx != 0 || dy != 0) camera.mouseLook(dx, dy);
    }

    private float axis(int posKey, int negKey) {
        float v = 0f;
        if (key(posKey)) v += 1f;
        if (key(negKey)) v -= 1f;
        return v;
    }

    public boolean isExitRequested() { return exitRequested; }

    public void captureMouse(boolean capture) {
        glfwSetInputMode(window, GLFW_CURSOR,
            capture ? GLFW_CURSOR_DISABLED : GLFW_CURSOR_NORMAL);
        if (glfwRawMouseMotionSupported())
            glfwSetInputMode(window, GLFW_RAW_MOUSE_MOTION,
                capture ? GLFW_TRUE : GLFW_FALSE);
        firstMouse = true;
    }

    private boolean key(int k) { return glfwGetKey(window, k) == GLFW_PRESS; }
}
