plugins {
  `maven-publish`
  signing
}

group = "com.github.android-password-store"
version = "0.1.5"

tasks.register<Exec>("buildGomobile") {
  commandLine("./build.sh", "android")
}

tasks.getByName("publish").dependsOn(tasks.getByName("buildGomobile"))

publishing {
  publications {
    create<MavenPublication>("gopenpgp") {
      artifactId = "gopenpgp"
      artifact("dist/android/Gopenpgp.aar")
      pom {
        name.set("gopenpgp")
        description.set("A high-level OpenPGP library")
        url.set("https://github.com/protonmail/gopenpgp")
        licenses {
          license {
            name.set("MIT")
            url.set("https://raw.githubusercontent.com/ProtonMail/gopenpgp/master/LICENSE")
          }
        }
        developers {
          developer {
            id.set("android-password-store")
            name.set("The Android Password Store Authors")
            email.set("aps@msfjarvis.dev")
          }
        }
        scm {
          connection.set("scm:git:https://github.com/Android-Password-Store/gopenpgp.git")
          developerConnection.set("scm:git:ssh://git@github.com:Android-Password-Store/gopenpgp.git")
          url.set("https://github.com/Android-Password-Store/gopenpgp")
        }
      }
    }
  }
  repositories {
    maven {
      val hostUrl = "https://oss.sonatype.org/"
      val releasesRepoUrl = uri("$hostUrl/service/local/staging/deploy/maven2/")
      val snapshotsRepoUrl = uri("$hostUrl/content/repositories/snapshots/")
      url = if (version.toString().endsWith("SNAPSHOT")) snapshotsRepoUrl else releasesRepoUrl
      credentials {
        username = findProperty("mavenUsername") as String?
        password = findProperty("mavenPassword") as String?
      }
    }
  }
}

signing {
  val signingKey: String? by project
  val signingPassword: String? by project
  useInMemoryPgpKeys(signingKey, signingPassword)
  sign(publishing.publications["gopenpgp"])
}
