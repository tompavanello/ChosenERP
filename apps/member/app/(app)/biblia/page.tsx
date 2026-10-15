"use client";

import { useEffect, useState } from "react";
import { ArrowLeft, BookText } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import {
  getBibleBooks,
  getBibleChapter,
  getBibleChapters,
  getBibleVersions,
  type BibleBook,
  type BibleChapterRef,
  type BiblePassage,
  type BibleVersion,
} from "@/lib/api";

export default function BibliaPage() {
  const [loading, setLoading] = useState(true);
  const [configured, setConfigured] = useState(true);
  const [versions, setVersions] = useState<BibleVersion[]>([]);
  const [version, setVersion] = useState<BibleVersion | null>(null);
  const [books, setBooks] = useState<BibleBook[] | null>(null);
  const [book, setBook] = useState<BibleBook | null>(null);
  const [chapters, setChapters] = useState<BibleChapterRef[] | null>(null);
  const [passage, setPassage] = useState<BiblePassage | null>(null);
  const [chapterLabel, setChapterLabel] = useState("");

  useEffect(() => {
    getBibleVersions()
      .then((d) => {
        setConfigured(d.configured);
        setVersions(d.versions ?? []);
        setVersion(d.versions?.[0] ?? null);
      })
      .catch(() => setConfigured(false))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!version) return;
    setBooks(null);
    getBibleBooks(version.id)
      .then(setBooks)
      .catch(() => setBooks([]));
  }, [version]);

  async function openBook(b: BibleBook) {
    if (!version) return;
    setBook(b);
    setChapters(null);
    setPassage(null);
    getBibleChapters(version.id, b.id)
      .then(setChapters)
      .catch(() => setChapters([]));
  }

  async function openChapter(c: BibleChapterRef) {
    if (!version || !book) return;
    setChapterLabel(`${book.name} ${c.number}`);
    setPassage(null);
    try {
      setPassage(await getBibleChapter(version.id, c.id));
    } catch {
      setPassage({ reference: `${book.name} ${c.number}`, content: "Não foi possível carregar o capítulo." });
    }
  }

  if (loading) return <SkeletonRows rows={4} />;

  if (!configured) {
    return (
      <div className="space-y-4">
        <h1 className="text-lg font-bold">Bíblia</h1>
        <Card>
          <p className="text-sm text-[var(--muted)]">
            O leitor da Bíblia ainda não está configurado pela igreja.
          </p>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <BookText className="h-5 w-5 text-[var(--brand)]" /> Bíblia
      </h1>

      <select
        value={version?.id ?? ""}
        onChange={(e) => {
          const v = versions.find((x) => x.id === e.target.value) ?? null;
          setVersion(v);
          setBook(null);
          setChapters(null);
          setPassage(null);
        }}
        className="w-full rounded-lg border border-[var(--line)] bg-[var(--card)] px-3 py-2 text-sm"
      >
        {versions.map((v) => (
          <option key={v.id} value={v.id}>
            {v.name} ({v.abbreviation})
          </option>
        ))}
      </select>

      {/* Capítulo aberto */}
      {passage && (
        <Card>
          <button
            onClick={() => setPassage(null)}
            className="mb-2 inline-flex items-center gap-1 text-xs text-[var(--muted)]"
          >
            <ArrowLeft className="h-3.5 w-3.5" /> voltar
          </button>
          <h2 className="text-base font-semibold">{passage.reference || chapterLabel}</h2>
          <p className="mt-2 whitespace-pre-line text-sm leading-relaxed">{passage.content}</p>
        </Card>
      )}

      {/* Capítulos do livro */}
      {!passage && book && (
        <Card>
          <button
            onClick={() => {
              setBook(null);
              setChapters(null);
            }}
            className="mb-2 inline-flex items-center gap-1 text-xs text-[var(--muted)]"
          >
            <ArrowLeft className="h-3.5 w-3.5" /> livros
          </button>
          <p className="mb-2 font-medium">{book.name}</p>
          {chapters === null ? (
            <SkeletonRows rows={2} />
          ) : (
            <div className="grid grid-cols-5 gap-2">
              {chapters.map((c) => (
                <button
                  key={c.id}
                  onClick={() => openChapter(c)}
                  className="rounded-lg bg-black/5 py-2 text-sm text-[var(--muted)]"
                >
                  {c.number}
                </button>
              ))}
            </div>
          )}
        </Card>
      )}

      {/* Livros */}
      {!passage && !book && (
        <Card>
          {books === null ? (
            <SkeletonRows rows={5} />
          ) : (
            <div className="grid grid-cols-3 gap-2">
              {books.map((b) => (
                <button
                  key={b.id}
                  onClick={() => openBook(b)}
                  className="truncate rounded-lg bg-black/5 px-2 py-2 text-sm text-[var(--muted)]"
                >
                  {b.name}
                </button>
              ))}
            </div>
          )}
        </Card>
      )}
    </div>
  );
}
