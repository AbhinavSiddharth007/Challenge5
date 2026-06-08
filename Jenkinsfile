pipeline {
    agent any


    parameters {
        string(
            name: 'EC2_HOST',
            defaultValue: '',
            description: 'Public IP (or DNS) of the EC2 instance to deploy to'
        )
    }

    environment {
        EC2_HOST     = "${params.EC2_HOST}"
        APP_PORT     = "4444"
        BINARY_NAME  = "myapp"
        SERVICE_NAME = "myapp"
    }

    stages {
        stage('Validate') {
            steps {
                script {
                    if (!params.EC2_HOST?.trim()) {
                        error 'EC2_HOST is empty — paste your EC2 public IP and re-run the build.'
                    }
                }
            }
        }

        stage('Build') {
            steps {
                echo 'Building a static linux/amd64 binary...'

                sh 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ${BINARY_NAME} .'
                sh 'file ${BINARY_NAME} || true'   // sanity-log the arch; harmless if `file` is absent
            }
        }

        stage('Deploy to EC2') {
            steps {
                withCredentials([
                    sshUserPrivateKey(
                        credentialsId: 'EC2_SSH_KEY',
                        keyFileVariable: 'KEY_FILE',
                        usernameVariable: 'SSH_USER'
                    )
                ]) {
                    sh """
                        # Copy binary to EC2
                        scp -i \$KEY_FILE \
                            -o StrictHostKeyChecking=no \
                            -o ConnectTimeout=15 \
                            ./${BINARY_NAME} \
                            \$SSH_USER@${EC2_HOST}:/tmp/${BINARY_NAME}

                        # SSH in: install binary, write systemd unit, start, and HEALTH-GATE the result
                        ssh -i \$KEY_FILE \
                            -o StrictHostKeyChecking=no \
                            -o ConnectTimeout=15 \
                            \$SSH_USER@${EC2_HOST} << 'REMOTE'
                                set -eu

                                # Install binary
                                sudo mv /tmp/${BINARY_NAME} /usr/local/bin/${BINARY_NAME}
                                sudo chmod +x /usr/local/bin/${BINARY_NAME}

                                # Write systemd unit file
                                sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null << 'UNIT'
[Unit]
Description=myapp service
After=network.target

[Service]
ExecStart=/usr/local/bin/${BINARY_NAME}
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
UNIT

                                # Reload systemd and (re)start service
                                sudo systemctl daemon-reload
                                sudo systemctl enable ${SERVICE_NAME}
                                sudo systemctl restart ${SERVICE_NAME}

                                # Health-gate: poll locally and FAIL the build if it never comes up.
                                # (Without this, an app that never starts still reports SUCCESS.)
                                ok=0
                                for i in \$(seq 1 12); do
                                    if curl -sf --max-time 5 http://localhost:${APP_PORT} >/dev/null; then
                                        ok=1
                                        break
                                    fi
                                    echo "Waiting for app on :${APP_PORT}... attempt \$i"
                                    sleep 5
                                done

                                if [ "\$ok" -ne 1 ]; then
                                    echo "ERROR: app never responded on localhost:${APP_PORT}"
                                    sudo systemctl status ${SERVICE_NAME} --no-pager || true
                                    sudo journalctl -u ${SERVICE_NAME} --no-pager -n 50 || true
                                    exit 1
                                fi

                                echo "App is up locally on port ${APP_PORT}"
REMOTE
                    """
                }
            }
        }

        stage('Verify (external)') {
            steps {
                // Curls the public IP from the Jenkins machine — the same path the dashboard grades.
                // Timeouts are critical: a Security Group DROP on :4444 would otherwise hang forever.
                sh """
                    for i in 1 2 3 4 5 6; do
                        if curl -sf --connect-timeout 5 --max-time 10 \
                                http://${EC2_HOST}:${APP_PORT} -o /tmp/verify_resp; then
                            echo 'External check OK:'
                            head -c 500 /tmp/verify_resp
                            echo
                            exit 0
                        fi
                        echo "External verify attempt \$i failed, retrying..."
                        sleep 5
                    done
                    echo 'ERROR: app not reachable externally on :${APP_PORT}.'
                    echo 'Hang/timeout points at a DROP (Security Group / NACL / host firewall).'
                    echo 'If it were "connection refused", suspect the app binding 127.0.0.1 instead of 0.0.0.0.'
                    exit 1
                """
            }
        }
    }

    post {
        success {
            echo "Deployment successful! App available at http://${EC2_HOST}:${APP_PORT}"
        }
        failure {
            echo 'Pipeline failed — check console output.'
        }
    }
}
