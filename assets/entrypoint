#!/usr/bin/python
# coding: utf-8

import os
import os.path
import subprocess

base = '/host'
pilot_fluentd = "fluentd"
pilot_filebeat = "filebeat"
ENV_PILOT_TYPE = "PILOT_TYPE"


def umount(volume):
    subprocess.check_call('umount -l %s' % volume, shell=True)


def mount_points():
    with open('/proc/self/mountinfo', 'r') as f:
        mounts = f.read().decode('utf-8')

    points = set()
    for line in mounts.split('\n'):
        mtab = line.split()
        if len(mtab) > 1 and mtab[4].startswith(base + '/') and mtab[4].endswith('shm') and 'containers' in mtab[4]:
            points.add(mtab[4])
    return points


def cleanup():
    umounts = mount_points()
    for volume in sorted(umounts, reverse=True):
        umount(volume)


def run():
    pilot_type = os.environ.get(ENV_PILOT_TYPE)
    if pilot_filebeat == pilot_type:
        tpl_config = "/pilot/filebeat.tpl"
    else:
        tpl_config = "/pilot/fluentd.tpl"

    os.execve('/pilot/pilot', ['/pilot/pilot', '-template', tpl_config, '-base', base, '-log-level', 'debug'],
              os.environ)


def config():
    pilot_type = os.environ.get(ENV_PILOT_TYPE)
    if pilot_filebeat == pilot_type:
        print "start log-pilot:", pilot_filebeat
        subprocess.check_call(['/pilot/config.filebeat'])
    else:
        print "start log-pilot:", pilot_fluentd
        subprocess.check_call(['/pilot/config.fluentd'])


if __name__ == '__main__':
    config()
    # FIXME need SYS_ADMIN capability
    cleanup()
    run()
