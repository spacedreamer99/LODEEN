package com.lodeen.engine.scene;

import com.lodeen.engine.graphics.Renderer3D;
import com.lodeen.engine.graphics.SkyboxRenderer;
import java.util.ArrayList;
import java.util.List;
import static org.lwjgl.opengl.GL11.*;

public class SceneRenderer {
    private final Renderer3D renderer;
    private final SkyboxRenderer skybox;
    private final RemotePlayerRenderer remotePlayers = new RemotePlayerRenderer();

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

        glDisable(GL_BLEND);
        glEnable(GL_CULL_FACE);
        glCullFace(GL_BACK);
        glDepthMask(true);
        renderer.render(opaque, camera, w, h);

        if (!transparent.isEmpty()) {
            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
            glDisable(GL_CULL_FACE);
            glDepthMask(false);
            renderer.render(transparent, camera, w, h);
            glEnable(GL_CULL_FACE);
            glDepthMask(true);
        }

        remotePlayers.render(camera, w, h);
    }
}
