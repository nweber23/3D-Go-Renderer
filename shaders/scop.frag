#version 410 core

in vec3 fragPos;
in vec3 vNormal;
in vec2 vUV;

uniform int hasNormals;   // 1: use attribute normals; 0: derive flat normals
uniform vec3 lightDir;    // world space, normalized
uniform sampler2D tex;
uniform float mixFactor;    // 0 = flat color, 1 = textured (smoothstepped)
uniform int wireframePass;  // 1: draw constant-color lines over the model

out vec4 fragColor;

void main() {
    if (wireframePass == 1) {
        fragColor = vec4(0.02, 0.02, 0.02, 1.0);
        return;
    }
    // Per-face grayscale driven by the primitive id: adjacent faces get
    // visibly different shades without duplicating any vertex data.
    float g = 0.3 + 0.6 * fract(float(gl_PrimitiveID) * 0.61803398875);

    // Flat face normal from screen-space derivatives when the mesh ships
    // no vn data. abs() keeps faces lit regardless of winding direction
    // (the raw teapot has inconsistent winding).
    vec3 n = (hasNormals == 1)
        ? normalize(vNormal)
        : normalize(cross(dFdx(fragPos), dFdy(fragPos)));
    float diff = abs(dot(n, lightDir)) * 0.65 + 0.35;

    vec3 flat_ = vec3(g) * diff;
    vec3 textured = texture(tex, vUV).rgb * diff;
    fragColor = vec4(mix(flat_, textured, mixFactor), 1.0);
}
