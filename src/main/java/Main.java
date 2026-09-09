import org.lwjgl.glfw.GLFW;
import org.lwjgl.glfw.GLFWErrorCallback;
import org.lwjgl.opengl.GL;
import org.lwjgl.system.MemoryUtil;

import static org.lwjgl.glfw.GLFW.*;
import static org.lwjgl.opengl.GL11.*;

public class Main {

    // Размеры окна
    private static final int WIDTH = 1280;
    private static final int HEIGHT = 720;
    private static final String TITLE = "LODEEN Engine";

    private long window;

    public static void main(String[] args) {
        new Main().run();
    }

    public void run() {
        init();
        loop();
        cleanup();
    }

    private void init() {
        // Установка обработчика ошибок GLFW
        GLFWErrorCallback.createPrint(System.err).set();

        // Инициализация GLFW
        if (!glfwInit()) {
            throw new IllegalStateException("Не удалось инициализировать GLFW");
        }

        // Настройки окна
        glfwDefaultWindowHints();
        glfwWindowHint(GLFW_VISIBLE, GLFW_FALSE);
        glfwWindowHint(GLFW_RESIZABLE, GLFW_TRUE);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MAJOR, 3);
        glfwWindowHint(GLFW_CONTEXT_VERSION_MINOR, 3);
        glfwWindowHint(GLFW_OPENGL_PROFILE, GLFW_OPENGL_CORE_PROFILE);

        // Создание окна
        window = glfwCreateWindow(WIDTH, HEIGHT, TITLE, MemoryUtil.NULL, MemoryUtil.NULL);
        if (window == MemoryUtil.NULL) {
            throw new RuntimeException("Не удалось создать окно GLFW");
        }

        // Установка контекста OpenGL
        glfwMakeContextCurrent(window);
        glfwSwapInterval(1); // Включаем вертикальную синхронизацию
        glfwShowWindow(window);

        // Загрузка возможностей OpenGL
        GL.createCapabilities();

        // Установка цвета очистки экрана (тёмно-синий, почти чёрный)
        glClearColor(0.0f, 0.0f, 0.1f, 1.0f);
    }

    private void loop() {
        // Главный цикл
        while (!glfwWindowShouldClose(window)) {
            // Очистка экрана
            glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);

            // Обмен буферов (показ кадра)
            glfwSwapBuffers(window);

            // Обработка событий (ввод, закрытие окна и т.д.)
            glfwPollEvents();
        }
    }

    private void cleanup() {
        // Уничтожение окна и завершение GLFW
        glfwDestroyWindow(window);
        glfwTerminate();

        // Освобождение обработчика ошибок
        GLFWErrorCallback callback = glfwSetErrorCallback(null);
        if (callback != null) {
            callback.free();
        }
    }
}