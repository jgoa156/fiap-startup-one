# -*- coding: utf-8 -*-
# Monta o video da apresentacao: cada pagina do PDF de slides fica na tela
# enquanto tocam, em sequencia, os audios gravados para ela.
#
#   python build-video.py
#
# Audios em ../Sound Recordings:
#   Intro.m4a          -> pagina 1 do PDF (a capa)
#   N.1.m4a, N.2.m4a.. -> pagina N+1 do PDF (o slide N)
# Para regravar um trecho, substitua o arquivo e rode de novo.
# Requer: pip install pymupdf imageio-ffmpeg

import os, re, sys, subprocess, tempfile, shutil
import pymupdf, imageio_ffmpeg

HERE = os.path.dirname(os.path.abspath(__file__))
PDF = os.path.join(HERE, 'Certamis-multicloud-slides.pdf')
AUD = os.path.join(os.path.dirname(HERE), 'Sound Recordings')
OUT = os.path.join(HERE, 'Certamis-multicloud-video.mp4')
FF = imageio_ffmpeg.get_ffmpeg_exe()


def duracao(arq):
    r = subprocess.run([FF, '-hide_banner', '-i', arq], capture_output=True, text=True)
    m = re.search(r'Duration: (\d+):(\d+):([\d.]+)', r.stderr)
    if not m:
        sys.exit('nao consegui ler a duracao de ' + arq)
    h, mi, s = m.groups()
    return int(h) * 3600 + int(mi) * 60 + float(s)


def roda(cmd):
    r = subprocess.run(cmd, capture_output=True, text=True)
    if r.returncode != 0:
        sys.stderr.write(r.stderr[-3000:])
        sys.exit('ffmpeg falhou')


# ---- agrupa os audios por pagina ------------------------------------------
grupos = {}
for nome in os.listdir(AUD):
    base, ext = os.path.splitext(nome)
    if ext.lower() not in ('.m4a', '.mp3', '.wav', '.aac', '.ogg'):
        continue
    if base.lower() == 'intro':
        grupos.setdefault(0, []).append((0, nome))
        continue
    m = re.fullmatch(r'(\d+)\.(\d+)', base)
    if not m:
        print('ignorado (nome fora do padrao):', nome)
        continue
    grupos.setdefault(int(m.group(1)), []).append((int(m.group(2)), nome))

doc = pymupdf.open(PDF)
paginas = len(doc)
faltam = [i for i in range(paginas) if i not in grupos]
sobram = [k for k in grupos if k >= paginas]
if faltam or sobram:
    sys.exit('audios nao batem com as paginas. Sem audio: %s. Sem pagina: %s' % (faltam, sobram))

tmp = tempfile.mkdtemp(prefix='video-')
segs = []
total = 0.0
print('pag  slide  audios                         duracao')
for i in range(paginas):
    arqs = [os.path.join(AUD, n) for _, n in sorted(grupos[i])]
    dur = sum(duracao(a) for a in arqs)
    total += dur

    # slide em 1920x1080
    png = os.path.join(tmp, 'p%02d.png' % i)
    pg = doc[i]
    z = 1920.0 / pg.rect.width
    pg.get_pixmap(matrix=pymupdf.Matrix(z, z)).save(png)

    seg = os.path.join(tmp, 's%02d.mp4' % i)
    cmd = [FF, '-y', '-hide_banner', '-loop', '1', '-framerate', '30', '-i', png]
    for a in arqs:
        cmd += ['-i', a]
    ent = ''.join('[%d:a]' % (k + 1) for k in range(len(arqs)))
    cmd += ['-filter_complex', ent + 'concat=n=%d:v=0:a=1,aresample=48000[a]' % len(arqs),
            '-map', '0:v', '-map', '[a]', '-t', '%.3f' % dur,
            '-c:v', 'libx264', '-preset', 'medium', '-tune', 'stillimage', '-pix_fmt', 'yuv420p',
            '-r', '30', '-c:a', 'aac', '-b:a', '160k', '-ac', '2', seg]
    roda(cmd)
    segs.append(seg)
    rot = 'capa' if i == 0 else str(i)
    print('%3d  %5s  %-30s %5.1f s' % (i + 1, rot, ', '.join(n for _, n in sorted(grupos[i])), dur))

lista = os.path.join(tmp, 'lista.txt')
with open(lista, 'w', encoding='utf-8') as f:
    for s in segs:
        f.write("file '%s'\n" % s.replace('\\', '/'))
roda([FF, '-y', '-hide_banner', '-f', 'concat', '-safe', '0', '-i', lista,
      '-c', 'copy', '-movflags', '+faststart', OUT])
shutil.rmtree(tmp, ignore_errors=True)

print('total: %d min %02d s' % (total // 60, total % 60))
print('video:', OUT, '(%d KB)' % (os.path.getsize(OUT) // 1024))
