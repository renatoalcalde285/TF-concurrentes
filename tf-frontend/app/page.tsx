"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

interface NodeSliderProps {
  nodeAmount: number;
  setNodeAmount: (value: number) => void;
}

const NodeSlider = ({ nodeAmount, setNodeAmount }: NodeSliderProps) => {
  return (
    <div className="flex flex-col items-center">
      <label htmlFor="node-slider" className="mb-2 text-lg font-semibold">
        Select Node Amount: {nodeAmount}
      </label>
      <input
        id="node-slider"
        type="range"
        min="1"
        max="5"
        value={nodeAmount}
        onChange={(e) => setNodeAmount(Number(e.target.value))}
        className="w-full h-2 bg-gray-300 rounded-lg appearance-none cursor-pointer accent-[#935AD8]"
      />
    </div>
  );
};

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
    <div className="flex flex-col gap-10">
      <NodeSlider nodeAmount={nodeAmount} setNodeAmount={setNodeAmount} />
      <input
        type="text"
        value={userId}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
        placeholder="Ingrese ID"
        className="py-3 px-5 rounded-3xl border-2 focus:border-[#CBC5EA] outline-none shadow-md text-center font-bold text-xl"
      />
    </div>
  );
}
