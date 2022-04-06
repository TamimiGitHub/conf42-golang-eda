// The following is an internal infrastructure file for building 
properties([
    buildDiscarder(logRotator(daysToKeepStr: '30', numToKeepStr: '10')),
])
currentBuild.rawBuild.getParent().setQuietPeriod(0)

library 'jenkins-pipeline-library@main'

stage('Build') {
  builder.goapi()
}