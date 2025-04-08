import { TypeAnimation as RawTypeAnimation } from "react-type-animation";
import "./TypeAnimation.css";

const TypeAnimation = ({
  sequence,
  wrapper = "span",
  cursor = true,
  repeat = Infinity,
}) => {
  return (
    <RawTypeAnimation
      sequence={sequence}
      wrapper={wrapper}
      cursor={cursor}
      repeat={repeat}
    />
  );
};

export default TypeAnimation;
