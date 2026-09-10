package com.lodeen.engine.scene;

import com.lodeen.engine.graphics.Renderer3D;
import com.lodeen.engine.graphics.SkyboxRenderer;
import java.util.ArrayList;
import java.util.List;
import static org.lwjgl.opengl.GL11.*;

public class SceneRenderer {
    private final Renderer3D renderer;
    private final SkyboxRenderer skybox;

    public SceneRenderer(Renderer3D renderer, SkyboxRenderer skybox) {
        this.renderer = renderer;
        this.skybox = skybox;
    }

    public void render(List<GameObject> objects, Camera camera, int w, int h) {
        glViewport(0, 0, w, h);
        glClear(GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT);
        skybox.render(camera, w, h);

        List<GameObject> opaque = new ArrayList<>();
        List<GameObject> transparent = new ArrayList<>();
        for (GameObject go : objects) {
            if (go.mesh == null) continue;
            if (go.isTransparent()) transparent.add(go);
            else opaque.add(go);
        }

        // Опаковые — с включённым culling (его включает Scene.init)
        glDisable(GL_BLEND);
        glDepthMask(true);
        renderer.render(opaque, camera, w, h);

        // Прозрачные — два прохода (back faces, потом front faces), без записи в depth
        if (!transparent.isEmpty()) {
            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
            glEnable(GL_CULL_FACE);
            glCullFace(GL_FRONT);
            glDepthMask(false);
            renderer.render(transparent, camera, w, h);
            glCullFace(GL_BACK);
            renderer.render(transparent, camera, w, h);
            glDepthMask(true);
        }
    }
}
