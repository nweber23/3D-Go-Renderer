#version 410 core

in vec3 fragPos;
in vec3 vNormal;
in vec2 vUV;

uniform int hasNormals;   // 1: use attribute normals; 0: derive flat normals
uniform vec3 lightDir;    // world space, normalized
uniform vec3 viewPos;     // camera eye, world space
uniform vec3 matKd;       // material diffuse (tints the grayscale scheme)
uniform vec3 matKs;       // material specular
uniform float matNs;      // specular exponent
uniform sampler2D tex;
uniform float mixFactor;    // 0 = flat color, 1 = textured (smoothstepped)
uniform int shadeMode;      // 0 faces, 1 normals, 2 uv checker
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

    // Blinn-Phong specular from the mtl Ks/Ns.
    vec3 h = normalize(lightDir + normalize(viewPos - fragPos));
    vec3 spec = matKs * pow(abs(dot(n, h)), max(matNs, 1.0)) * 0.35;

    // Kd tints the per-face grayscale so mtl colors show while faces stay
    // distinguishable; scaled by 1/0.64 so the default gray keeps the
    // original brightness.
    if (shadeMode == 1) {
        fragColor = vec4(n * 0.5 + 0.5, 1.0);
        return;
    }
    if (shadeMode == 2) {
        vec2 c = floor(vUV * 16.0);
        float chk = mod(c.x + c.y, 2.0);
        fragColor = vec4(vec3(mix(0.15, 0.95, chk)) * diff + spec, 1.0);
        return;
    }
    vec3 flat_ = matKd * (g / 0.64) * diff + spec;
    vec3 textured = texture(tex, vUV).rgb * diff + spec;
    fragColor = vec4(mix(flat_, textured, mixFactor), 1.0);
}
