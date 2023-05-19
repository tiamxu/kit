package config

type Config struct {
	Jenkins `yaml:"jenkins"`
	Gitlab  `yaml:"gitlab"`
	DB      `yaml:"db"`
}
type Jenkins struct {
	Url      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Gitlab struct {
	Url   string `yaml:"url"`
	Token string `yaml:"token"`
}
type DB struct {
	Driver          string `yaml:"driver"`
	Database        string `yaml:"database"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	MaxIdleConns    int    `yaml:"max_idle_conns"`
	MaxOpenConns    int    `yaml:"max_open_conns"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
}

const JenkinsJobConfig = `<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job@1292.v27d8cc3e2602">
<actions>
  <org.jenkinsci.plugins.pipeline.modeldefinition.actions.DeclarativeJobAction plugin="pipeline-model-definition@2.2131.vb_9788088fdb_5"/>
  <org.jenkinsci.plugins.pipeline.modeldefinition.actions.DeclarativeJobPropertyTrackerAction plugin="pipeline-model-definition@2.2131.vb_9788088fdb_5">
	<jobProperties/>
	<triggers/>
	<parameters/>
	<options/>
  </org.jenkinsci.plugins.pipeline.modeldefinition.actions.DeclarativeJobPropertyTrackerAction>
</actions>
<description></description>
<keepDependencies>false</keepDependencies>
<properties/>
<definition class="org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition" plugin="workflow-cps@3659.v582dc37621d8">
  <script>node{
	  stage(&apos;Loading&apos;)
	  def rootDir = pwd()
	  println(rootDir)
	  def pipeline = load &apos;pipeline.groovy&apos;
	  pipeline(&apos;${app_name}&apos;,&apos;${app_group}&apos;)
}</script>
  <sandbox>true</sandbox>
</definition>
<triggers/>
<disabled>false</disabled>
</flow-definition>`
