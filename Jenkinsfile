pipeline {
    agent any

    environment {
        IMAGE = "ttl.sh/abhinavsiddharth007:2h"
        KUBE_API = "https://kubernetes.default.svc"
    }

    stages {
        stage('Lint') {
            steps {
                echo 'Linting pod.yaml...'
                sh 'kubectl apply --dry-run=client -f pod.yaml'
            }
        }

        stage('Build & Push') {
            steps {
                echo "Image: ${IMAGE} (already pushed in previous challenge)"
            }
        }

        stage('Deploy') {
            steps {
                withCredentials([string(credentialsId: 'KUBECONFIG_TOKEN', variable: 'TOKEN')]) {
                    sh """
                        kubectl --server=${KUBE_API} \
                                --token=\$TOKEN \
                                --insecure-skip-tls-verify=true \
                                apply -f pod.yaml

                        kubectl --server=${KUBE_API} \
                                --token=\$TOKEN \
                                --insecure-skip-tls-verify=true \
                                wait pod/myapp \
                                --for=condition=Ready \
                                --timeout=120s
                    """
                }
            }
        }
    }

    post {
        success {
            echo 'Pod myapp is Running!'
        }
        failure {
            echo 'Pipeline failed — check console output.'
        }
    }
}
