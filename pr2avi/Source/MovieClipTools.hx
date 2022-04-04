package;

import openfl.geom.ColorTransform;
import openfl.display.MovieClip;

class MovieClipTools {
    public static inline function get(mc:MovieClip, name:String):MovieClip {
        return cast(mc.getChildByName(name), MovieClip);
    }

    public static inline function setColor(mc:MovieClip, color:Int):Void {
        var ct = new ColorTransform();
        ct.color = color;
        mc.transform.colorTransform = ct;
    }
}