pipeline {
    agent any

    environment {
        // Public IP of your EC2 instance — update this after provisioning
        EC2_HOST = "YOUR_EC2_PUBLIC_IP"
        EC2_USER = "ubuntu"                   // use "ec2-user" for Amazon Linux
        APP_PORT = "4444"
        BINARY_NAME = "myapp"
        SERVICE_NAME = "myapp"
    }

    stages {
        stage('Build') {
            steps {
                echo 'Building the application...'
                sh 'go build -o ${BINARY_NAME} .'
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
                            ./${BINARY_NAME} \
                            \$SSH_USER@${EC2_HOST}:/tmp/${BINARY_NAME}

                        # SSH in, install binary, configure systemd, start service
                        ssh -i \$KEY_FILE \
                            -o StrictHostKeyChecking=no \
                            \$SSH_USER@${EC2_HOST} << 'REMOTE'
                                set -e

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

                                # Wait for the app to be up
                                for i in \$(seq 1 12); do
                                    curl -sf http://localhost:${APP_PORT} && break
                                    echo "Waiting for app... attempt \$i"
                                    sleep 5
                                done

                                echo "App is running on port ${APP_PORT}"
REMOTE
                    """
                }
            }
        }

        stage('Verify') {
            steps {
                sh "curl -sf http://${EC2_HOST}:${APP_PORT} | head -c 500"
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