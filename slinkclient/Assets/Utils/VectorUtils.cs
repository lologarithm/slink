using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using UnityEngine;

namespace Assets.Utils
{
    public class VectorUtils
    {
        public static Vector2 RotateVect2(Vector2 v, float degrees)
        {
            Vector2 result = new Vector2();
            result.x = (int)(v.x * (float)Math.Cos(degrees) - v.y * (float)Math.Sin(degrees));
            result.y = (int)(v.x * (float)Math.Sin(degrees) + v.y * (float)Math.Cos(degrees));
            return result;
        }

        public static Vect2 GameToNetworkVect2(Vector2 v) {
            return new Vect2() { X = (int)v.x, Y = (int)v.y };
        }

        public static Vector2 NetworkToGameVect2(Vect2 v)
        {
            return new Vector2(v.X, v.Y);
        }
    }
}
