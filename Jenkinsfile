// throttle concurrent build
properties([[$class: 'ThrottleJobProperty', categories: [], limitOneJobWithMatchingParams: false, maxConcurrentPerNode: 1, maxConcurrentTotal: 1, paramsToUseForLimit: '', throttleEnabled: true, throttleOption: 'project']])

node('build') {
    stage "Checkout"
    // git checkout
    checkout scm
    // git update submodules
    sh 'git submodule update --init --recursive'

    stage "Build"
    sh "./build/run.sh hack/build-go.sh"

    stage "Docker"
    def dockerTag = env.BRANCH_NAME.replaceAll("/", "_")
    // docker build
	dir('cluster/images/hyperkube') {
    	sh "make ARCH=amd64 REGISTRY=visenze VERSION=${dockerTag} build"
	}
    // docker push
    retry(10) {
        sh "docker push visenze/hyperkube-amd64:${dockerTag}"
    }
}