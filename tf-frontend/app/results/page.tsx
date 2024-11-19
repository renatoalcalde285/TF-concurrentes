// ResultsPage.tsx

"use client";

import { useSearchParams, useRouter } from "next/navigation";
import React, { useEffect, useState } from "react";

interface Recommendation {
  MovieID: string;
  Rating: number;
}

export const ResultCard = ({
  movieId,
  score,
}: {
  movieId: string;
  score: number;
}) => {
  return (
    <div className="flex flex-col gap-2 rounded-xl p-6 bg-white border border-[#CBC5EA]">
      <p className="text-xl lg:text-3xl font-bold">Película: {movieId}</p>
      <p className="text-lg lg:text-2xl">Score: {score.toFixed(2)}</p>
    </div>
  );
};

export default function ResultsPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const userId = searchParams.get("userId");

  useEffect(() => {
    const fetchRecommendations = async () => {
      try {
        const response = await fetch(
          `http://localhost:8080/recommend?userId=${userId}`
        );
        if (!response.ok) {
          throw new Error("Failed to fetch recommendations");
        }
        const data = await response.json();
        setRecommendations(data.recommendations);
      } catch (error) {
        console.error("Error fetching recommendations:", error);
      }
    };

    if (userId) {
      fetchRecommendations();
    }
  }, [userId]);

  const handleReturn = () => {
    router.push("/");
  };
  //Here
  return (
    <div className="flex flex-col items-center gap-16 w-2/3">
      <h1 className="text-3xl lg:text-4xl">
        Bienvenid@ <span className="font-bold">{userId}</span>, te recomendamos:
      </h1>
      <div className="flex justify-center w-full gap-6">
        {recommendations.map((rec) => (
          <ResultCard
            key={rec.MovieID}
            movieId={rec.MovieID}
            score={rec.Rating}
          />
        ))}
      </div>
      <div
        className="bg-[#CBC5EA] p-4 rounded-full border self-start hover:cursor-pointer hover:shadow-[0_0_6px_6px_rgba(203,197,234,0.6)] transition-shadow duration-300"
        onClick={handleReturn}
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="24"
          height="24"
          fill="none"
          stroke="#4a4a4a"
          strokeWidth="2.5"
          strokeLinecap="round"
          viewBox="0 0 16 16"
        >
          <path d="M11.354 1.646a.5.5 0 0 1 0 .708L5.707 8l5.647 5.646a.5.5 0 0 1-.708.708l-6-6a.5.5 0 0 1 0-.708l6-6a.5.5 0 0 1 .708 0" />
        </svg>
      </div>
    </div>
  );
}
