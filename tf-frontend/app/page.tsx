"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

export default function HomePage() {
  const router = useRouter();
  const [userId, setUserId] = useState("");
  const [nodeAmount, setNodeAmount] = useState(3);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setUserId(e.target.value);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" && userId.length > 0) {
      router.push(`/results?userId=${userId}&nodeAmount=${nodeAmount}`);
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-3xl font-bold lg:text-4xl">
         ¡Tu próxima película favorita te espera!
      </h1>
      <input
        type="text"
        value={userId}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder="Ingrese su userID"
        className="py-3 px-5 rounded-3xl border-2 focus:border-[#CBC5EA] outline-none shadow-md text-center font-bold text-xl"
      />
    </div>
  );
}
