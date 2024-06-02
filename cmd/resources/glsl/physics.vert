#version 440 core

uniform mat4 modelViewProjectionMatrix;
uniform mat4 modelViewMatrix;

in layout(location = 0) vec3 position;

uniform vec3 color;
uniform float alpha;

out vec4 color4;

void main() {
    gl_Position = modelViewProjectionMatrix * modelViewMatrix * vec4(position, 1.0);

    color4 = vec4(color, alpha);
}
