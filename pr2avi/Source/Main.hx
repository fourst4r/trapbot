package;


import haxe.crypto.Md5;
import openfl.system.System;
import openfl.display.PNGEncoderOptions;
import openfl.utils.Future;
import openfl.geom.Matrix;
import openfl.geom.Rectangle;
import openfl.display.JPEGEncoderOptions;
import haxe.io.BytesOutput;
import openfl.display.MovieClip;
import openfl.display.Bitmap;
import openfl.display.BitmapData;
import openfl.display.Sprite;
import openfl.Assets;

using MovieClipTools;

class Main extends Sprite {
	
	
	public function new () {
		
		super ();

		var mc = Assets.getMovieClip("pr2:CharacterGraphic");
		


		while (true) {
			try {

				var line = #if cpp Sys.args()[0]; #else Sys.stdin().readLine(); #end

				trace(line);
				if (line == "quit")
					break;

				var s = line.split("_").map(a -> {
					var parsed = Std.parseInt(a);
					return parsed == null ? 0 : parsed;
				});

				trace(s);
				
				var img = generate(mc, s[0], s[1], s[2], s[3], s[4], s[5], 
					s[6], s[7], s[8], s[9], s[10], s[11]);
	
				line = Md5.encode(line);

				var f = sys.io.File.write('$line.png');
				f.writeBytes(img, 0, img.length);
				f.close();
				
				#if cpp
				break;
				#else
				Sys.stdout().writeString("done");
				#end
					
			} catch (e:haxe.io.Eof) {
				break;
			} catch (e) {

				//Sys.stderr().writeString(e.toString());
				#if neko neko.Lib.rethrow (e); #else break; #end
			}
		}

		System.exit(0);
	}

	function generate(mc:MovieClip, 
		hat, hatC1, hatC2, 
		head, headC1, headC2, 
		body, bodyC1, bodyC2,
		feet, feetC1, feetC2) {

		mc.stop();

		var anim = mc.get("standAnim");
		anim.x = 50;
		anim.y = 120;
		anim.scaleX = -.25;
		anim.scaleY = .25;
		
		anim.get("weapon").gotoAndStop(0);

		anim.get("head").visible = (body != 29);
		anim.get("head").gotoAndStop(head);
		anim.get("head").get("colorMC").gotoAndStop(head);
		anim.get("head").get("colorMC").setColor(headC1);
		anim.get("head").get("colorMC2").gotoAndStop(head);
		anim.get("head").get("colorMC2").setColor(headC2);

		var headmc = anim.get("head");

		headmc.get("hat1").gotoAndStop(hat);
		headmc.get("hat1").get("colorMC").gotoAndStop(hat);
		headmc.get("hat1").get("colorMC").setColor(hatC1);
		headmc.get("hat1").get("colorMC2").gotoAndStop(hat);
		headmc.get("hat1").get("colorMC2").setColor(hatC2);

		headmc.get("hat2").gotoAndStop(0);
		headmc.get("hat2").get("colorMC").gotoAndStop(0);
		headmc.get("hat2").get("colorMC2").gotoAndStop(0);

		headmc.get("hat3").gotoAndStop(0);
		headmc.get("hat3").get("colorMC").gotoAndStop(0);
		headmc.get("hat3").get("colorMC2").gotoAndStop(0);

		headmc.get("hat4").gotoAndStop(0);
		headmc.get("hat4").get("colorMC").gotoAndStop(0);
		headmc.get("hat4").get("colorMC2").gotoAndStop(0);

		anim.get("body").gotoAndStop(body);
		anim.get("body").get("colorMC").gotoAndStop(body);
		anim.get("body").get("colorMC").setColor(bodyC1);
		anim.get("body").get("colorMC2").gotoAndStop(body);
		anim.get("body").get("colorMC2").setColor(bodyC2);
		
		anim.get("foot1").visible = (body != 29);
		anim.get("foot1").gotoAndStop(feet);
		anim.get("foot1").get("colorMC").gotoAndStop(feet);
		anim.get("foot1").get("colorMC").setColor(feetC1);
		anim.get("foot1").get("colorMC2").gotoAndStop(feet);
		anim.get("foot1").get("colorMC2").setColor(feetC2);
		
		anim.get("foot2").visible = (body != 29);
		anim.get("foot2").gotoAndStop(feet);
		anim.get("foot2").get("colorMC").gotoAndStop(feet);
		anim.get("foot2").get("colorMC").setColor(feetC1);
		anim.get("foot2").get("colorMC2").gotoAndStop(feet);
		anim.get("foot2").get("colorMC2").setColor(feetC2);

		var bmpdata = new BitmapData(90, 130, true, 0);
		bmpdata.draw(anim, anim.transform.matrix);
		var img = bmpdata.encode(new Rectangle(0, 0, 90, 130), new PNGEncoderOptions());
		
		return img;

	}
	
	
}