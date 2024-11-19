// ResultsPage.tsx

"use client";

import { useSearchParams, useRouter } from "next/navigation";
import React, { useEffect, useState } from "react";
import Papa from "papaparse";

interface Recommendation {
  MovieID: string;
  Rating: number;
}

export const ResultCard = ({
  movieName,
  score,
}: {
  movieName: string;
  score: number;
}) => {
  return (
    <div className="flex flex-col gap-2 rounded-xl p-6 bg-white border border-[#CBC5EA] flex-1">
      <p className="text-lg lg:text-3xl font-bold"> {movieName}</p>
      <p className="text-lg lg:text-2xl mt-auto">Score: {score.toFixed(2)}</p>
    </div>
  );
};

export default function ResultsPage() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [errorMessage, setErrorMessage] = useState<string>("");
  const [movieMap, setMovieMap] = useState<{ [key: string]: string }>({});
  const userId = searchParams.get("userId");

  const [movieDataLoaded, setMovieDataLoaded] = useState<boolean>(false);
  const [recommendationsLoaded, setRecommendationsLoaded] =
    useState<boolean>(false);

  useEffect(() => {
    const fetchMovieData = async () => {
      try {
        const response = await fetch("/movieData.csv");
        const csvData = await response.text();

        Papa.parse(csvData, {
          delimiter: ",",
          skipEmptyLines: true,
          complete: (results) => {
            const data = results.data as string[][];
            const map: { [key: string]: string } = {};
            data.forEach((row) => {
              const [movieID, year, name] = row;
              map[movieID] = name;
            });
            setMovieMap(map);
            setMovieDataLoaded(true);
          },
          error: (error: any) => {
            console.error("Error parsing CSV:", error);
            setErrorMessage("Error loading movie data.");
            setLoading(false);
          },
        });
      } catch (error) {
        console.error("Error fetching movie data:", error);
        setErrorMessage("Error fetching movie data.");
        setLoading(false);
      }
    };

    fetchMovieData();
  }, []);

  useEffect(() => {
    const fetchRecommendations = async () => {
      try {
        const response = await fetch(
          `http://localhost:8080/recommend?userId=${userId}`
        );
        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText || "Failed to fetch recommendations");
        }
        const data = await response.json();
        setRecommendations(data.recommendations);
        setRecommendationsLoaded(true);
      } catch (error: any) {
        console.error("Error fetching recommendations:", error);
        setErrorMessage(
          error.message ||
            "Lo sentimos, no pudimos obtener las recomendaciones. Por favor, inténtalo de nuevo."
        );
        setLoading(false);
      }
    };

    if (userId) {
      fetchRecommendations();
    }
  }, [userId]);

  useEffect(() => {
    if (movieDataLoaded && recommendationsLoaded) {
      setLoading(false);
    }
  }, [movieDataLoaded, recommendationsLoaded]);

  const handleReturn = () => {
    router.push("/");
  };

  return (
    <div className="flex flex-col items-center gap-16 w-2/3">
      {!errorMessage && (
        <h1 className="text-3xl lg:text-4xl">
          Bienvenid@ <span className="font-bold">{userId}</span>,
          {!loading && " te recomendamos:"}
        </h1>
      )}
      {loading ? (
        <p className="text-xl lg:text-2xl">
          Estamos buscando recomendaciones para ti...
        </p>
      ) : errorMessage ? (
        <p className="text-red-500 text-xl lg:text-2xl">
          No pudimos encontrar a <b>{userId}</b> en nuestra base de datos,
          porfavor probar con otro userID
        </p>
      ) : (
        <div className="flex justify-center w-full gap-6">
          {recommendations.map((rec) => (
            <ResultCard
              key={rec.MovieID}
              movieName={movieMap[rec.MovieID] || rec.MovieID}
              score={rec.Rating}
            />
          ))}
        </div>
      )}
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
