+++
date = '2018-06-18T00:00:00Z'
tags = ['git', 'docker', 'bash', 'windows']
title = 'Fixing Docker volume paths on Git Bash on Windows'

+++

My normal development laptop runs Windows, but like a lot of developers, I make huge use of Docker, which I run under Hyper-V.  I also heavily use the git bash terminal on windows to work.

Usually, everything works as expected, but I was recently trying to run an ELK (Elasticsearch, Logstash, Kibana) container, and needed to pass in an extra configuration file for Logstash.  This caused me a lot of trouble, as nothing was working as expected.

The command I was running is as follows:

```bash
docker run \
    -d --rm \
    --name elk_temp \
    -p 5044:5044 \
    -p 5601:5601 \
    -p 9200:9200 \
    -v logstash/app.conf:/etc/logstash/conf.d/app.conf \
    sebp/elk
```

But this has the interesting effect of mounting the `app.conf` in the container as a directory (which is empty), rather than doing the useful thing of mounting it as a file. Hmm.  I realised it was git bash doing path transformations to the windows style causing the issue, but all the work arounds I tried failed:


```bash
# single quotes
docker run ... -v 'logstash/app.conf:/etc/logstash/conf.d/app.conf'
# absolute path
docker run ... -v /d/dev/temp/logstash/app.conf:/etc/logstash/conf.d/app.conf
# absolute path with // prefix
docker run ... -v //d/dev/temp/logstash/app.conf:/etc/logstash/conf.d/app.conf
```

In the end, I found a way to switch off MSYS's (what git bash is based on) path conversion:


```bash
MSYS_NO_PATHCONV=1 docker run \
    -d --rm \
    --name elk_temp \
    -p 5044:5044 \
    -p 5601:5601 \
    -p 9200:9200 \
    -v logstash/app.conf:/etc/logstash/conf.d/app.conf \
    sebp/elk
```

And Voila, the paths get passed through correctly, and I can go back to hacking away at Logstash!