{{/*
common.copyScripts — initContainer that decodes base64-embedded scripts from
the chart's scripts/ directory into a shared /scripts emptyDir volume.

Usage: include this template at the Pod spec level (under spec.template.spec
for Jobs, or spec for bare Pods). It produces:
  - initContainers: block with the copy-scripts container
  - volumes: block with the scripts emptyDir

The consuming template must add its own containers: block with a volumeMount
for the "scripts" volume at /scripts. Example:

    spec:
      {{- include "common.copyScripts" . | nindent 6 }}
      containers:
        - name: my-task
          image: bitnami/kubectl:latest
          command: ["/scripts/my-script.sh"]
          volumeMounts:
            - name: scripts
              mountPath: /scripts
*/}}
{{- define "common.copyScripts" }}
initContainers:
  - name: copy-scripts
    image: busybox:stable
    command:
      - /bin/sh
      - -c
      - |
        {{- range $path, $content := .Files.Glob "scripts/*.sh" }}
        echo "{{ $content | toString | b64enc }}" | base64 -d > /scripts/{{ base $path }}
        chmod +x /scripts/{{ base $path }}
        {{- end }}
    volumeMounts:
      - name: scripts
        mountPath: /scripts
volumes:
  - name: scripts
    emptyDir: {}
{{- end }}