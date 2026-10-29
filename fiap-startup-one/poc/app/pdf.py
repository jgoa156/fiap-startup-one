"""PDF mínimo, só com a biblioteca padrão: uma página, texto em Helvetica.

Basta para o cartão de confirmação da POC. O que importa no ensaio não é a
beleza do PDF, e sim que ele seja gerado fora do request e servido como arquivo.
"""


def _escapar(texto: str) -> bytes:
    b = texto.encode("cp1252", errors="replace")
    return b.replace(b"\\", b"\\\\").replace(b"(", b"\\(").replace(b")", b"\\)")


def pagina(linhas: list[tuple[int, str]]) -> bytes:
    """linhas: (tamanho_da_fonte, texto), de cima para baixo."""
    conteudo = [b"BT", b"50 790 Td"]
    for tamanho, texto in linhas:
        conteudo.append(b"/F1 %d Tf" % tamanho)
        conteudo.append(b"(" + _escapar(texto) + b") Tj")
        conteudo.append(b"0 -%d Td" % int(tamanho * 1.6))
    conteudo.append(b"ET")
    fluxo = b"\n".join(conteudo)

    objetos = [
        b"<< /Type /Catalog /Pages 2 0 R >>",
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] "
        b"/Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
        b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
        b"<< /Length %d >>\nstream\n" % len(fluxo) + fluxo + b"\nendstream",
    ]
    saida = bytearray(b"%PDF-1.4\n")
    posicoes = []
    for i, obj in enumerate(objetos, start=1):
        posicoes.append(len(saida))
        saida += b"%d 0 obj\n" % i + obj + b"\nendobj\n"
    xref = len(saida)
    saida += b"xref\n0 %d\n0000000000 65535 f \n" % (len(objetos) + 1)
    for p in posicoes:
        saida += b"%010d 00000 n \n" % p
    saida += b"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n" % (len(objetos) + 1, xref)
    return bytes(saida)
